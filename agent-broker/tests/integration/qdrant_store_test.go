package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/store"
)

var testHost string

func TestMain(m *testing.M) {
	_ = godotenv.Load("../../.env")

	testHost = os.Getenv("QDRANT_HOST")
	if testHost == "" {
		fmt.Println("QDRANT_HOST not set, skipping integration tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func setupStore(t *testing.T) *store.QdrantStore {
	t.Helper()
	ctx := context.Background()

	collectionName := "test_" + uuid.New().String()[:8]
	s, err := store.NewQdrantStore(ctx,
		store.WithHost(testHost),
		store.WithCollectionName(collectionName),
		store.WithVectorDimension(4),
	)
	if err != nil {
		t.Fatalf("failed to create QdrantStore: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return s
}

func validAgentCard() a2a.AgentCard {
	return a2a.AgentCard{
		Name:        "Test Agent",
		Description: "A test agent",
		URL:         "http://localhost:9000",
		Version:     "1.0.0",
		Skills: []a2a.AgentSkill{
			{ID: "skill-1", Name: "Skill One"},
		},
	}
}

func validAgent(id string) *store.RegisteredAgent {
	now := time.Now()
	return &store.RegisteredAgent{
		ID:        id,
		Card:      validAgentCard(),
		Tags:      []string{"test"},
		Embedding: []float32{0.1, 0.2, 0.3, 0.4},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestQdrantStore_CreateAgent(t *testing.T) {
	t.Parallel()

	t.Run("creates new agent", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		err := s.CreateAgent(ctx, validAgent("agent-1"))

		if err != nil {
			t.Errorf("CreateAgent() error = %v, want nil", err)
		}
	})

	t.Run("duplicate ID returns ErrAlreadyExists", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		_ = s.CreateAgent(ctx, validAgent("agent-1"))
		err := s.CreateAgent(ctx, validAgent("agent-1"))

		if err != store.ErrAlreadyExists {
			t.Errorf("CreateAgent() error = %v, want ErrAlreadyExists", err)
		}
	})
}

func TestQdrantStore_GetAgent(t *testing.T) {
	t.Parallel()

	t.Run("returns existing agent", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		original := validAgent("agent-1")
		_ = s.CreateAgent(ctx, original)

		agent, err := s.GetAgent(ctx, "agent-1")

		if err != nil {
			t.Fatalf("GetAgent() error = %v, want nil", err)
		}
		if agent.ID != "agent-1" {
			t.Errorf("GetAgent() ID = %v, want agent-1", agent.ID)
		}
		if agent.Card.Name != original.Card.Name {
			t.Errorf("GetAgent() Card.Name = %v, want %v", agent.Card.Name, original.Card.Name)
		}
	})

	t.Run("non-existent returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		_, err := s.GetAgent(ctx, "not-exists")

		if err != store.ErrNotFound {
			t.Errorf("GetAgent() error = %v, want ErrNotFound", err)
		}
	})
}

func TestQdrantStore_ListAgents(t *testing.T) {
	t.Parallel()

	t.Run("empty store returns zero agents", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		result, err := s.ListAgents(ctx, store.AgentFilter{Limit: 10})

		if err != nil {
			t.Fatalf("ListAgents() error = %v", err)
		}
		if len(result.Agents) != 0 {
			t.Errorf("ListAgents() got %d agents, want 0", len(result.Agents))
		}
		if result.Total != 0 {
			t.Errorf("ListAgents() total = %d, want 0", result.Total)
		}
	})

	t.Run("pagination offset and limit", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			agent := validAgent(fmt.Sprintf("agent-%d", i))
			agent.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
			_ = s.CreateAgent(ctx, agent)
		}

		result, err := s.ListAgents(ctx, store.AgentFilter{Offset: 1, Limit: 2})

		if err != nil {
			t.Fatalf("ListAgents() error = %v", err)
		}
		if len(result.Agents) != 2 {
			t.Errorf("ListAgents() got %d agents, want 2", len(result.Agents))
		}
		if result.Total != 5 {
			t.Errorf("ListAgents() total = %d, want 5", result.Total)
		}
	})

	t.Run("filter by tags", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		agent1 := validAgent("agent-1")
		agent1.Tags = []string{"prod", "ml"}
		agent2 := validAgent("agent-2")
		agent2.Tags = []string{"dev"}
		_ = s.CreateAgent(ctx, agent1)
		_ = s.CreateAgent(ctx, agent2)

		result, err := s.ListAgents(ctx, store.AgentFilter{Tags: []string{"prod"}, Limit: 10})

		if err != nil {
			t.Fatalf("ListAgents() error = %v", err)
		}
		if len(result.Agents) != 1 {
			t.Errorf("ListAgents() got %d agents, want 1", len(result.Agents))
		}
		if result.Agents[0].ID != "agent-1" {
			t.Errorf("ListAgents() got ID = %v, want agent-1", result.Agents[0].ID)
		}
	})

	t.Run("filter by skills", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		agent1 := validAgent("agent-1")
		agent1.Card.Skills = []a2a.AgentSkill{{ID: "translate", Name: "Translate"}}
		agent2 := validAgent("agent-2")
		agent2.Card.Skills = []a2a.AgentSkill{{ID: "summarize", Name: "Summarize"}}
		_ = s.CreateAgent(ctx, agent1)
		_ = s.CreateAgent(ctx, agent2)

		result, err := s.ListAgents(ctx, store.AgentFilter{Skills: []string{"translate"}, Limit: 10})

		if err != nil {
			t.Fatalf("ListAgents() error = %v", err)
		}
		if len(result.Agents) != 1 {
			t.Errorf("ListAgents() got %d agents, want 1", len(result.Agents))
		}
		if result.Agents[0].ID != "agent-1" {
			t.Errorf("ListAgents() got ID = %v, want agent-1", result.Agents[0].ID)
		}
	})
}

func TestQdrantStore_UpdateAgent(t *testing.T) {
	t.Parallel()

	t.Run("updates existing agent", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		original := validAgent("agent-1")
		_ = s.CreateAgent(ctx, original)

		updated := validAgent("agent-1")
		updated.Card.Name = "Updated Name"
		err := s.UpdateAgent(ctx, updated)

		if err != nil {
			t.Fatalf("UpdateAgent() error = %v, want nil", err)
		}

		agent, _ := s.GetAgent(ctx, "agent-1")
		if agent.Card.Name != "Updated Name" {
			t.Errorf("GetAgent() Card.Name = %v, want Updated Name", agent.Card.Name)
		}
	})

	t.Run("non-existent returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		err := s.UpdateAgent(ctx, validAgent("not-exists"))

		if err != store.ErrNotFound {
			t.Errorf("UpdateAgent() error = %v, want ErrNotFound", err)
		}
	})
}

func TestQdrantStore_DeleteAgent(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing agent", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		_ = s.CreateAgent(ctx, validAgent("agent-1"))

		err := s.DeleteAgent(ctx, "agent-1")

		if err != nil {
			t.Fatalf("DeleteAgent() error = %v, want nil", err)
		}

		_, err = s.GetAgent(ctx, "agent-1")
		if err != store.ErrNotFound {
			t.Errorf("GetAgent() after delete should return ErrNotFound, got %v", err)
		}
	})

	t.Run("non-existent returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		err := s.DeleteAgent(ctx, "not-exists")

		if err != store.ErrNotFound {
			t.Errorf("DeleteAgent() error = %v, want ErrNotFound", err)
		}
	})
}

func TestQdrantStore_SearchAgents(t *testing.T) {
	t.Parallel()

	t.Run("empty store returns zero results", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		query := []float32{1.0, 0.0, 0.0, 0.0}
		result, err := s.SearchAgents(ctx, query, 10, store.AgentFilter{})

		if err != nil {
			t.Fatalf("SearchAgents() error = %v", err)
		}
		if len(result.Agents) != 0 {
			t.Errorf("SearchAgents() got %d agents, want 0", len(result.Agents))
		}
	})

	t.Run("returns agents sorted by similarity score", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		agent1 := validAgent("agent-high")
		agent1.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
		agent2 := validAgent("agent-mid")
		agent2.Embedding = []float32{0.7, 0.7, 0.0, 0.0}
		agent3 := validAgent("agent-low")
		agent3.Embedding = []float32{0.0, 1.0, 0.0, 0.0}
		_ = s.CreateAgent(ctx, agent1)
		_ = s.CreateAgent(ctx, agent2)
		_ = s.CreateAgent(ctx, agent3)

		query := []float32{1.0, 0.0, 0.0, 0.0}
		result, err := s.SearchAgents(ctx, query, 10, store.AgentFilter{})

		if err != nil {
			t.Fatalf("SearchAgents() error = %v", err)
		}
		if len(result.Agents) != 3 {
			t.Fatalf("SearchAgents() got %d agents, want 3", len(result.Agents))
		}
		if result.Agents[0].Agent.ID != "agent-high" {
			t.Errorf("SearchAgents() first result ID = %v, want agent-high", result.Agents[0].Agent.ID)
		}
		if result.Agents[0].Score < result.Agents[1].Score {
			t.Errorf("SearchAgents() results not sorted by score descending")
		}
		if result.Agents[1].Score < result.Agents[2].Score {
			t.Errorf("SearchAgents() results not sorted by score descending")
		}
	})

	t.Run("filter by tags with vector search", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		agent1 := validAgent("agent-1")
		agent1.Tags = []string{"prod"}
		agent1.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
		agent2 := validAgent("agent-2")
		agent2.Tags = []string{"dev"}
		agent2.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
		_ = s.CreateAgent(ctx, agent1)
		_ = s.CreateAgent(ctx, agent2)

		query := []float32{1.0, 0.0, 0.0, 0.0}
		result, err := s.SearchAgents(ctx, query, 10, store.AgentFilter{Tags: []string{"prod"}})

		if err != nil {
			t.Fatalf("SearchAgents() error = %v", err)
		}
		if len(result.Agents) != 1 {
			t.Errorf("SearchAgents() got %d agents, want 1", len(result.Agents))
		}
		if result.Agents[0].Agent.ID != "agent-1" {
			t.Errorf("SearchAgents() got ID = %v, want agent-1", result.Agents[0].Agent.ID)
		}
	})

	t.Run("filter by skills with vector search", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		agent1 := validAgent("agent-1")
		agent1.Card.Skills = []a2a.AgentSkill{{ID: "translate", Name: "Translate"}}
		agent1.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
		agent2 := validAgent("agent-2")
		agent2.Card.Skills = []a2a.AgentSkill{{ID: "summarize", Name: "Summarize"}}
		agent2.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
		_ = s.CreateAgent(ctx, agent1)
		_ = s.CreateAgent(ctx, agent2)

		query := []float32{1.0, 0.0, 0.0, 0.0}
		result, err := s.SearchAgents(ctx, query, 10, store.AgentFilter{Skills: []string{"translate"}})

		if err != nil {
			t.Fatalf("SearchAgents() error = %v", err)
		}
		if len(result.Agents) != 1 {
			t.Errorf("SearchAgents() got %d agents, want 1", len(result.Agents))
		}
		if result.Agents[0].Agent.ID != "agent-1" {
			t.Errorf("SearchAgents() got ID = %v, want agent-1", result.Agents[0].Agent.ID)
		}
	})

	t.Run("limit parameter respected", func(t *testing.T) {
		t.Parallel()
		s := setupStore(t)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			agent := validAgent(fmt.Sprintf("agent-%d", i))
			agent.Embedding = []float32{1.0, 0.0, 0.0, 0.0}
			_ = s.CreateAgent(ctx, agent)
		}

		query := []float32{1.0, 0.0, 0.0, 0.0}
		result, err := s.SearchAgents(ctx, query, 2, store.AgentFilter{})

		if err != nil {
			t.Fatalf("SearchAgents() error = %v", err)
		}
		if len(result.Agents) != 2 {
			t.Errorf("SearchAgents() got %d agents, want 2", len(result.Agents))
		}
	})
}
