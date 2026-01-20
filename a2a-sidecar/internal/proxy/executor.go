package proxy

import (
	"context"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
)

// proxyExecutor implements [a2asrv.AgentExecutor], which is a required [a2asrv.RequestHandler] dependency.
// It is responsible for proxying the request to a backend agent, translating the response to a2a.Event objects and writing them to the provided [eventqueue.Queue].
type proxyExecutor struct{}

var _ a2asrv.AgentExecutor = (*proxyExecutor)(nil)

func NewProxyExecutor() *proxyExecutor {
	return &proxyExecutor{}
}

func (*proxyExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	response := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: "Hello, world!"})
	return q.Write(ctx, response)
}

func (*proxyExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	return nil
}
