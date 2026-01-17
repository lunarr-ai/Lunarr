package store

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
)

// MemoryStore implements AgentStore with in-memory storage.
type MemoryStore struct {
	// mu protects agents map.
	mu sync.RWMutex
	// agents is the in-memory agent storage.
	agents map[string]*RegisteredAgent
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents: make(map[string]*RegisteredAgent),
	}
}

// Ping always returns nil for memory store.
func (s *MemoryStore) Ping(_ context.Context) error {
	return nil
}

// Close is a no-op for memory store.
func (s *MemoryStore) Close() error {
	return nil
}

// CreateAgent stores a new agent.
func (s *MemoryStore) CreateAgent(_ context.Context, agent *RegisteredAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[agent.ID]; exists {
		return ErrAlreadyExists
	}

	s.agents[agent.ID] = agent
	return nil
}

// GetAgent retrieves an agent by ID.
func (s *MemoryStore) GetAgent(_ context.Context, id string) (*RegisteredAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[id]
	if !exists {
		return nil, ErrNotFound
	}

	return agent, nil
}

// ListAgents returns agents matching the filter.
func (s *MemoryStore) ListAgents(_ context.Context, filter AgentFilter) (*AgentListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*RegisteredAgent
	for _, agent := range s.agents {
		if matchesFilter(agent, filter) {
			filtered = append(filtered, agent)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := len(filtered)

	start := min(filter.Offset, len(filtered))
	end := min(start+filter.Limit, len(filtered))

	return &AgentListResult{
		Agents: filtered[start:end],
		Total:  total,
	}, nil
}

// UpdateAgent updates an existing agent.
func (s *MemoryStore) UpdateAgent(_ context.Context, agent *RegisteredAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[agent.ID]; !exists {
		return ErrNotFound
	}

	s.agents[agent.ID] = agent
	return nil
}

// DeleteAgent removes an agent.
func (s *MemoryStore) DeleteAgent(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[id]; !exists {
		return ErrNotFound
	}

	delete(s.agents, id)
	return nil
}

// SearchAgents finds agents by vector similarity with optional filtering.
func (s *MemoryStore) SearchAgents(_ context.Context, query []float32, limit int, filter AgentFilter) (*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scored []ScoredAgent
	for _, agent := range s.agents {
		if !matchesFilter(agent, filter) {
			continue
		}
		if len(agent.Embedding) == 0 {
			continue
		}

		score := cosineSimilarity(query, agent.Embedding)
		scored = append(scored, ScoredAgent{
			Agent: agent,
			Score: score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	return &SearchResult{Agents: scored}, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func matchesFilter(agent *RegisteredAgent, filter AgentFilter) bool {
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, t := range filter.Tags {
			for _, at := range agent.Tags {
				if t == at {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	if len(filter.Skills) > 0 {
		hasSkill := false
		for _, s := range filter.Skills {
			for _, skill := range agent.Card.Skills {
				if s == skill.ID {
					hasSkill = true
					break
				}
			}
			if hasSkill {
				break
			}
		}
		if !hasSkill {
			return false
		}
	}

	if filter.Query != "" {
		query := strings.ToLower(filter.Query)
		if !strings.Contains(strings.ToLower(agent.Card.Name), query) &&
			!strings.Contains(strings.ToLower(agent.Card.Description), query) {
			return false
		}
	}

	return true
}
