package tools

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
)

// NewBroadcastTool creates a tool for broadcasting to multiple agents.
func NewBroadcastTool(reg *registry.RegistryService) (tool.Tool, error) {
	return functiontool.New(
		functiontool.Config{
			Name:        "broadcast",
			Description: "Find multiple agents to broadcast a request to. Use this when a task should be sent to several relevant agents.",
		},
		func(ctx tool.Context, args BroadcastArgs) (BroadcastResult, error) {
			limit := args.Limit
			if limit <= 0 {
				limit = 5
			}

			result, err := reg.Discover(ctx, registry.DiscoverInput{
				Query:  args.Query,
				Limit:  limit,
				Tags:   args.Tags,
				Skills: args.Skills,
			})
			if err != nil {
				return BroadcastResult{}, err
			}

			agents := make([]ScoredAgent, 0, len(result.Agents))
			for _, scored := range result.Agents {
				agents = append(agents, ScoredAgent{
					Card:  scored.Agent.Card,
					Score: scored.Score,
				})
			}

			return BroadcastResult{
				Agents: agents,
				Total:  len(agents),
			}, nil
		},
	)
}
