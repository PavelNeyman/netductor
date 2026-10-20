package httpx

import (
	"net/http"
	"time"
)

// NewServer is the short-request listener (API, not a deploy stream).
// WriteTimeout is 60s. Long streams must use their own Server (op serve uses 2h).
func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
