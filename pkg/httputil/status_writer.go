package httputil

import "net/http"

// StatusWriter captures the HTTP status code written by handlers.
type StatusWriter struct {
	http.ResponseWriter
	Status int
}

func NewStatusWriter(w http.ResponseWriter) *StatusWriter {
	return &StatusWriter{ResponseWriter: w, Status: http.StatusOK}
}

func (w *StatusWriter) WriteHeader(code int) {
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap supports http.ResponseController and similar wrappers.
func (w *StatusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
