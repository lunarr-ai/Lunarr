package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
)

// A2ATranslator implements Translator for A2A-compliant backends.
// It acts as a passthrough proxy since both sides speak the A2A protocol.
type A2ATranslator struct {
	// backendURL is the URL of the backend A2A agent endpoint.
	backendURL string
}

var _ Translator = (*A2ATranslator)(nil)

// NewA2ATranslator creates a new A2ATranslator.
func NewA2ATranslator(backendURL string) *A2ATranslator {
	return &A2ATranslator{backendURL: backendURL}
}

func (t *A2ATranslator) TranslateRequest(ctx context.Context, method string, params any) (*http.Request, error) {
	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.backendURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// For streaming methods, set Accept header for SSE
	if method == "message/stream" || method == "tasks/resubscribe" {
		req.Header.Set("Accept", "text/event-stream")
	}

	return req, nil
}

func (t *A2ATranslator) TranslateResponse(ctx context.Context, resp *http.Response, q eventqueue.Queue) error {
	defer func() { _ = resp.Body.Close() }()

	contentType := resp.Header.Get("Content-Type")

	// Handle streaming response (SSE)
	if strings.HasPrefix(contentType, "text/event-stream") {
		return t.handleSSEResponse(ctx, resp, q)
	}

	// Handle non-streaming JSON-RPC response
	return t.handleJSONResponse(ctx, resp, q)
}

func (t *A2ATranslator) handleSSEResponse(ctx context.Context, resp *http.Response, q eventqueue.Queue) error {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		event, final, err := t.parseSSEEvent(data)
		if err != nil {
			continue // Skip malformed events
		}

		if err := q.Write(ctx, event.(a2a.Event)); err != nil {
			return err
		}

		if final {
			break
		}
	}
	return scanner.Err()
}

func (t *A2ATranslator) handleJSONResponse(ctx context.Context, resp *http.Response, q eventqueue.Queue) error {
	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *jsonRPCError   `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("backend error: %s", rpcResp.Error.Message)
	}

	event, _, err := t.parseEventFromResult(rpcResp.Result)
	if err != nil {
		return err
	}

	return q.Write(ctx, event.(a2a.Event))
}

// jsonRPCError represents a JSON-RPC error response.
type jsonRPCError struct {
	// Code is the error code.
	Code int `json:"code"`
	// Message is the error message.
	Message string `json:"message"`
	// Data is optional additional error data.
	Data any `json:"data,omitempty"`
}

func (t *A2ATranslator) parseSSEEvent(data string) (any, bool, error) {
	var rpcResp struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(data), &rpcResp); err != nil {
		return nil, false, err
	}
	return t.parseEventFromResult(rpcResp.Result)
}

func (t *A2ATranslator) parseEventFromResult(result json.RawMessage) (any, bool, error) {
	var kindCheck struct {
		Kind  string `json:"kind"`
		Final bool   `json:"final"`
	}
	if err := json.Unmarshal(result, &kindCheck); err != nil {
		return nil, false, err
	}

	var event any
	var err error

	switch kindCheck.Kind {
	case "task":
		var task a2a.Task
		err = json.Unmarshal(result, &task)
		event = &task
	case "message":
		var msg a2a.Message
		err = json.Unmarshal(result, &msg)
		event = &msg
	case "status-update":
		var e a2a.TaskStatusUpdateEvent
		err = json.Unmarshal(result, &e)
		event = &e
	case "artifact-update":
		var e a2a.TaskArtifactUpdateEvent
		err = json.Unmarshal(result, &e)
		event = &e
	default:
		return nil, false, errors.New("unknown event kind: " + kindCheck.Kind)
	}

	return event, kindCheck.Final, err
}
