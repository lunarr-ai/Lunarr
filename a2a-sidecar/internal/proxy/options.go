package proxy

import (
	"log/slog"
	"net/http"
	"time"
)

// Options holds configurable values for the proxyExecutor.
type Options struct {
	// HTTPClient is the HTTP client for backend requests.
	HTTPClient *http.Client
	// Timeout is the request timeout duration.
	Timeout time.Duration
	// Logger is the structured logger.
	Logger *slog.Logger
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
		Timeout: 300 * time.Second,
		Logger:  slog.Default(),
	}
}

// Option is a functional option for configuring the proxyExecutor.
type Option func(*Options)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) { o.HTTPClient = client }
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.Timeout = timeout }
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) { o.Logger = logger }
}
