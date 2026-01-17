package tools

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
)

// NewRouteTool creates a tool for routing to the best matching agent.
func NewRouteTool(reg *registry.RegistryService) (tool.Tool, error) {
	return functiontool.New(
		functiontool.Config{
			Name:        "route",
			Description: "Find the single best agent for a task. Use this when you need to forward a request to the most relevant agent.",
		},
		func(ctx tool.Context, args RouteArgs) (RouteResult, error) {
			result, err := reg.Discover(ctx, registry.DiscoverInput{
				Query:  args.Query,
				Limit:  1,
				Tags:   args.Tags,
				Skills: args.Skills,
			})
			if err != nil {
				return RouteResult{}, err
			}

			if len(result.Agents) == 0 {
				return RouteResult{Found: false}, nil
			}

			agent := result.Agents[0]
			return RouteResult{
				Agent: &ScoredAgent{
					Card:  agent.Agent.Card,
					Score: agent.Score,
				},
				Found: true,
			}, nil
		},
	)
}
