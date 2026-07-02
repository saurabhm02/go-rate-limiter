package ratelimitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Decision is the parsed response from POST /v1/check.
type Decision struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	ResetAt    int64
	RetryAfter int64
	Algorithm  string
}

// Client calls the production rate limiter over HTTP.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type checkRequest struct {
	Route string `json:"route"`
	Cost  int64  `json:"cost"`
}

type checkResponse struct {
	Allowed    bool   `json:"allowed"`
	Limit      int64  `json:"limit,omitempty"`
	Remaining  int64  `json:"remaining,omitempty"`
	ResetAt    int64  `json:"reset_at,omitempty"`
	RetryAfter *int64 `json:"retry_after,omitempty"`
	Algorithm  string `json:"algorithm,omitempty"`
}

// Check asks the rate limiter whether a route is allowed.
func (c *Client) Check(ctx context.Context, route string, cost int64) (Decision, error) {
	if cost <= 0 {
		cost = 1
	}

	body, err := json.Marshal(checkRequest{Route: route, Cost: cost})
	if err != nil {
		return Decision{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/check", bytes.NewReader(body))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Decision{}, err
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		return Decision{}, fmt.Errorf("rate limiter unavailable")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return Decision{}, fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return Decision{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed checkResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Decision{}, err
	}

	dec := Decision{
		Allowed:   parsed.Allowed,
		Limit:     parsed.Limit,
		Remaining: parsed.Remaining,
		ResetAt:   parsed.ResetAt,
		Algorithm: parsed.Algorithm,
	}
	if parsed.RetryAfter != nil {
		dec.RetryAfter = *parsed.RetryAfter
	}
	if dec.RetryAfter == 0 {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				dec.RetryAfter = n
			}
		}
	}

	return dec, nil
}
