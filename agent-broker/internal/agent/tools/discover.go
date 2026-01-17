package tools

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
)

// NewDiscoverTool creates a tool for discovering agents by semantic search.
func NewDiscoverTool(reg *registry.RegistryService) (tool.Tool, error) {
	return functiontool.New(
		functiontool.Config{
			Name:        "discover",
			Description: "Find agents matching a natural language query. Returns a list of agents ranked by relevance.",
		},
		func(ctx tool.Context, args DiscoverArgs) (DiscoverResult, error) {
			limit := args.Limit
			if limit <= 0 {
				limit = 10
			}

			result, err := reg.Discover(ctx, registry.DiscoverInput{
				Query:  args.Query,
				Limit:  limit,
				Tags:   args.Tags,
				Skills: args.Skills,
			})
			if err != nil {
				return DiscoverResult{}, err
			}

			agents := make([]ScoredAgent, 0, len(result.Agents))
			for _, scored := range result.Agents {
				agents = append(agents, ScoredAgent{
					Card:  scored.Agent.Card,
					Score: scored.Score,
				})
			}

			return DiscoverResult{
				Agents: agents,
				Total:  len(agents),
			}, nil
		},
	)
}
