package tools

import "github.com/a2aproject/a2a-go/a2a"

// DiscoverArgs are the arguments for the discover tool.
type DiscoverArgs struct {
	// Query is the natural language search query.
	Query string `json:"query"`
	// Limit is the maximum number of results to return.
	Limit int `json:"limit,omitempty"`
	// Tags filters by classification tags.
	Tags []string `json:"tags,omitempty"`
	// Skills filters by skill IDs.
	Skills []string `json:"skills,omitempty"`
}

// RouteArgs are the arguments for the route tool.
type RouteArgs struct {
	// Query is the natural language search query.
	Query string `json:"query"`
	// Tags filters by classification tags.
	Tags []string `json:"tags,omitempty"`
	// Skills filters by skill IDs.
	Skills []string `json:"skills,omitempty"`
}

// BroadcastArgs are the arguments for the broadcast tool.
type BroadcastArgs struct {
	// Query is the natural language search query.
	Query string `json:"query"`
	// Limit is the maximum number of agents to broadcast to.
	Limit int `json:"limit,omitempty"`
	// Tags filters by classification tags.
	Tags []string `json:"tags,omitempty"`
	// Skills filters by skill IDs.
	Skills []string `json:"skills,omitempty"`
}

// ScoredAgent represents an agent with a relevance score.
type ScoredAgent struct {
	// Card is the agent's A2A card.
	Card a2a.AgentCard `json:"card"`
	// Score is the relevance score.
	Score float32 `json:"score"`
}

// DiscoverResult is the result of the discover tool.
type DiscoverResult struct {
	// Agents is the list of matching agents with scores.
	Agents []ScoredAgent `json:"agents"`
	// Total is the total number of results.
	Total int `json:"total"`
}

// RouteResult is the result of the route tool.
type RouteResult struct {
	// Agent is the best matching agent.
	Agent *ScoredAgent `json:"agent,omitempty"`
	// Found indicates whether a matching agent was found.
	Found bool `json:"found"`
}

// BroadcastResult is the result of the broadcast tool.
type BroadcastResult struct {
	// Agents is the list of agents to broadcast to.
	Agents []ScoredAgent `json:"agents"`
	// Total is the total number of agents.
	Total int `json:"total"`
}
