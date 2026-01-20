package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
)

// methodMapping maps handler method names to JSON-RPC method names.
var methodMapping = map[string]string{
	"OnSendMessage":       "message/send",
	"OnSendMessageStream": "message/stream",
	"OnGetTask":           "tasks/get",
	"OnCancelTask":        "tasks/cancel",
	"OnResubscribeToTask": "tasks/resubscribe",
}

// proxyExecutor implements [a2asrv.AgentExecutor], which is a required [a2asrv.RequestHandler] dependency.
// It is responsible for proxying the request to a backend agent, translating the response to a2a.Event objects and writing them to the provided [eventqueue.Queue].
type proxyExecutor struct {
	// translator handles request/response translation.
	translator Translator
	// httpClient is the HTTP client for backend communication.
	httpClient *http.Client
	// timeout is the request timeout duration.
	timeout time.Duration
	// logger is the structured logger.
	logger *slog.Logger
}

var _ a2asrv.AgentExecutor = (*proxyExecutor)(nil)

// NewProxyExecutor creates a new proxyExecutor with the given translator and options.
func NewProxyExecutor(translator Translator, opts ...Option) *proxyExecutor {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return &proxyExecutor{
		translator: translator,
		httpClient: options.HTTPClient,
		timeout:    options.Timeout,
		logger:     options.Logger,
	}
}

func (p *proxyExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	method, err := p.getJSONRPCMethod(ctx)
	if err != nil {
		return err
	}

	p.logger.Debug("proxying request",
		"method", method,
		"task_id", reqCtx.TaskID,
	)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	params := p.buildParams(reqCtx)

	req, err := p.translator.TranslateRequest(ctx, method, params)
	if err != nil {
		return fmt.Errorf("failed to translate request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("backend request failed: %w", err)
	}

	return p.translator.TranslateResponse(ctx, resp, q)
}

func (p *proxyExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	p.logger.Info("cancelling task", "task_id", reqCtx.TaskID)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	params := &a2a.TaskIDParams{ID: reqCtx.TaskID}

	req, err := p.translator.TranslateRequest(ctx, "tasks/cancel", params)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}

	return p.translator.TranslateResponse(ctx, resp, q)
}

func (p *proxyExecutor) getJSONRPCMethod(ctx context.Context) (string, error) {
	callCtx, ok := a2asrv.CallContextFrom(ctx)
	if !ok {
		return "", fmt.Errorf("call context not found")
	}

	handlerMethod := callCtx.Method()
	jsonRPCMethod, ok := methodMapping[handlerMethod]
	if !ok {
		return "", fmt.Errorf("unknown handler method: %s", handlerMethod)
	}

	return jsonRPCMethod, nil
}

func (p *proxyExecutor) buildParams(reqCtx *a2asrv.RequestContext) *a2a.MessageSendParams {
	return &a2a.MessageSendParams{
		Message: reqCtx.Message,
	}
}
