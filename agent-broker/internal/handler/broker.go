package handler

import (
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adka2a"
	"google.golang.org/adk/session"
)

// BrokerHandler handles A2A protocol requests using the ADK executor.
type BrokerHandler struct {
	// handler is the a2asrv handler that wraps the executor.
	handler a2asrv.RequestHandler
	// agentCard is the broker's A2A agent card.
	agentCard *a2a.AgentCard
}

// NewBrokerHandler creates a new A2A handler with the given agent and session service.
func NewBrokerHandler(brokerAgent agent.Agent, sessionService session.Service) *BrokerHandler {
	executor := adka2a.NewExecutor(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        brokerAgent.Name(),
			Agent:          brokerAgent,
			SessionService: sessionService,
		},
	})

	handler := a2asrv.NewHandler(executor)

	// Build agent card from the agent
	skills := adka2a.BuildAgentSkills(brokerAgent)
	card := &a2a.AgentCard{
		Name:        brokerAgent.Name(),
		Description: brokerAgent.Description(),
		Version:     "1.0.0",
		Skills:      skills,
	}

	return &BrokerHandler{
		handler:   handler,
		agentCard: card,
	}
}

// RegisterRoutes registers A2A routes on the given ServeMux.
func (h *BrokerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /", a2asrv.NewJSONRPCHandler(h.handler))
	mux.Handle("GET /.well-known/agent-card.json", a2asrv.NewStaticAgentCardHandler(h.agentCard))
}
