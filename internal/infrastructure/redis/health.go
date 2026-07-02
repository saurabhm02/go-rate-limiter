package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type ClientPinger struct {
	client *goredis.Client
}

func NewClientPinger(client *goredis.Client) ClientPinger {
	return ClientPinger{client: client}
}

func (p ClientPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
