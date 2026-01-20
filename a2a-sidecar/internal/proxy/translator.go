package proxy

import (
	"context"
	"net/http"

	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
)

// Translator translates between the A2A protocol and backend-specific formats.
type Translator interface {
	// TranslateRequest converts an A2A request to a backend-specific HTTP request.
	// method is the JSON-RPC method name (e.g., "message/send", "message/stream").
	// params is the request parameters (e.g., *a2a.MessageSendParams).
	TranslateRequest(ctx context.Context, method string, params any) (*http.Request, error)

	// TranslateResponse reads the backend response and writes events to the queue.
	// For streaming (SSE), it parses events and writes each to the queue.
	// For non-streaming, it writes the single result to the queue.
	TranslateResponse(ctx context.Context, resp *http.Response, q eventqueue.Queue) error
}
