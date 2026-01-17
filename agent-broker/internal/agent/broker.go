package agent

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/agent/tools"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
)

// Broker metadata.
const (
	brokerName        = "Lunarr Agent Broker"
	brokerDescription = "A2A-compliant meta-agent for agent discovery, routing, and broadcast"
)

const brokerInstruction = `You are the Lunarr Agent Broker, a meta-agent that helps users discover, route to, and broadcast requests to other agents in the network.

You have three capabilities:
1. **discover**: Find agents matching a query. Use this to show users available agents for a topic.
2. **route**: Find the single best agent for a specific task. Use this when a user needs to be directed to one agent.
3. **broadcast**: Find multiple agents to send a request to. Use this when a task should go to several agents.

When users describe what they need, use the appropriate tool to find matching agents. Be helpful and explain the results clearly.`

// Options configures the broker agent.
type Options struct {
	// GeminiAPIKey is the API key for Gemini.
	GeminiAPIKey string
	// GeminiModel is the model name to use.
	GeminiModel string
}

// DefaultOptions returns sensible defaults for broker options.
func DefaultOptions() Options {
	return Options{
		GeminiModel: "gemini-3-flash-preview",
	}
}

// Option is a functional option for configuring the broker agent.
type Option func(*Options)

// WithGeminiAPIKey sets the Gemini API key.
func WithGeminiAPIKey(key string) Option {
	return func(o *Options) {
		o.GeminiAPIKey = key
	}
}

// WithGeminiModel sets the Gemini model name.
func WithGeminiModel(model string) Option {
	return func(o *Options) {
		if model != "" {
			o.GeminiModel = model
		}
	}
}

// NewBrokerAgent creates a new ADK LLM agent for the broker.
func NewBrokerAgent(ctx context.Context, reg *registry.RegistryService, opts ...Option) (agent.Agent, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	model, err := gemini.NewModel(ctx, options.GeminiModel, &genai.ClientConfig{
		APIKey: options.GeminiAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini model: %w", err)
	}

	discoverTool, err := tools.NewDiscoverTool(reg)
	if err != nil {
		return nil, fmt.Errorf("create discover tool: %w", err)
	}

	routeTool, err := tools.NewRouteTool(reg)
	if err != nil {
		return nil, fmt.Errorf("create route tool: %w", err)
	}

	broadcastTool, err := tools.NewBroadcastTool(reg)
	if err != nil {
		return nil, fmt.Errorf("create broadcast tool: %w", err)
	}

	return llmagent.New(llmagent.Config{
		Name:        brokerName,
		Description: brokerDescription,
		Model:       model,
		Instruction: brokerInstruction,
		Tools:       []tool.Tool{discoverTool, routeTool, broadcastTool},
	})
}

// Name returns the broker agent name.
func Name() string {
	return brokerName
}

// Description returns the broker agent description.
func Description() string {
	return brokerDescription
}

// NewSessionService returns an in-memory session service for the broker.
func NewSessionService() session.Service {
	return session.InMemoryService()
}
