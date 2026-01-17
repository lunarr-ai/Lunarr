package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// Options configures the QdrantStore.
type Options struct {
	// Host is the Qdrant server hostname.
	Host string
	// Port is the Qdrant gRPC port.
	Port int
	// APIKey is the optional API key for authentication.
	APIKey string
	// UseTLS enables TLS for the connection.
	UseTLS bool
	// CollectionName is the name of the Qdrant collection for agents.
	CollectionName string
	// VectorDimension is the size of embedding vectors.
	VectorDimension uint64
}

// DefaultOptions returns Options with sensible defaults.
// Note: VectorDimension must be set explicitly via WithVectorDimension().
func DefaultOptions() Options {
	return Options{
		Host:           "localhost",
		Port:           6334,
		APIKey:         "",
		UseTLS:         false,
		CollectionName: "agents",
	}
}

// Option is a functional option for configuring QdrantStore.
type Option func(*Options)

// WithHost sets the Qdrant server hostname.
func WithHost(host string) Option {
	return func(o *Options) {
		o.Host = host
	}
}

// WithPort sets the Qdrant gRPC port.
func WithPort(port int) Option {
	return func(o *Options) {
		o.Port = port
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(apiKey string) Option {
	return func(o *Options) {
		o.APIKey = apiKey
	}
}

// WithTLS enables TLS for the connection.
func WithTLS(useTLS bool) Option {
	return func(o *Options) {
		o.UseTLS = useTLS
	}
}

// WithCollectionName sets the Qdrant collection name.
func WithCollectionName(name string) Option {
	return func(o *Options) {
		o.CollectionName = name
	}
}

// WithVectorDimension sets the embedding vector dimension.
func WithVectorDimension(dim uint64) Option {
	return func(o *Options) {
		o.VectorDimension = dim
	}
}

// QdrantStore implements Store using Qdrant as the vector database.
type QdrantStore struct {
	// client is the Qdrant gRPC client.
	client *qdrant.Client
	// collectionName is the name of the agents collection.
	collectionName string
}

// NewQdrantStore creates a QdrantStore with the given options.
// VectorDimension must be set via WithVectorDimension().
func NewQdrantStore(ctx context.Context, opts ...Option) (*QdrantStore, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if options.VectorDimension == 0 {
		return nil, fmt.Errorf("VectorDimension must be set via WithVectorDimension()")
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   options.Host,
		Port:   options.Port,
		APIKey: options.APIKey,
		UseTLS: options.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	store := &QdrantStore{
		client:         client,
		collectionName: options.CollectionName,
	}

	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	if err := store.ensureCollection(ctx, options); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist.
func (s *QdrantStore) ensureCollection(ctx context.Context, opts Options) error {
	exists, err := s.client.CollectionExists(ctx, opts.CollectionName)
	if err != nil {
		return fmt.Errorf("check collection exists: %w", err)
	}

	if exists {
		return nil
	}

	err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: opts.CollectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     opts.VectorDimension,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	// Create payload indexes for efficient filtering
	// Index on agent ID for lookups
	keywordIndexes := []string{"id", "tags", "skill_ids"}
	for _, field := range keywordIndexes {
		_, err = s.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: opts.CollectionName,
			FieldName:      field,
			FieldType:      qdrant.PtrOf(qdrant.FieldType_FieldTypeKeyword),
		})
		if err != nil {
			return fmt.Errorf("create %s index: %w", field, err)
		}
	}

	textIndexes := []string{"card_name", "card_description"}
	for _, field := range textIndexes {
		_, err = s.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: opts.CollectionName,
			FieldName:      field,
			FieldType:      qdrant.PtrOf(qdrant.FieldType_FieldTypeText),
		})
		if err != nil {
			return fmt.Errorf("create %s index: %w", field, err)
		}
	}

	// Create integer index with range support for ordering
	_, err = s.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: opts.CollectionName,
		FieldName:      "created_at",
		FieldType:      qdrant.PtrOf(qdrant.FieldType_FieldTypeInteger),
		FieldIndexParams: &qdrant.PayloadIndexParams{
			IndexParams: &qdrant.PayloadIndexParams_IntegerIndexParams{
				IntegerIndexParams: &qdrant.IntegerIndexParams{
					Lookup: qdrant.PtrOf(true),
					Range:  qdrant.PtrOf(true),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create created_at index: %w", err)
	}

	return nil
}

// Ping checks if Qdrant is reachable and healthy.
func (s *QdrantStore) Ping(ctx context.Context) error {
	_, err := s.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("qdrant health check failed: %w", err)
	}
	return nil
}

// Close closes the Qdrant client connection.
func (s *QdrantStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// CreateAgent stores a new agent in Qdrant.
func (s *QdrantStore) CreateAgent(ctx context.Context, agent *RegisteredAgent) error {
	// Check if agent already exists by searching payload
	existing, err := s.findPointByAgentID(ctx, agent.ID)
	if err != nil {
		return fmt.Errorf("check agent exists: %w", err)
	}
	if existing != nil {
		return ErrAlreadyExists
	}

	payload, err := agentToPayload(agent)
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	// Generate random UUID for point ID
	pointID := uuid.New().String()

	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewID(pointID),
				Vectors: qdrant.NewVectorsDense(agent.Embedding),
				Payload: payload,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("upsert point: %w", err)
	}

	return nil
}

// findPointByAgentID searches for a point by agent ID in payload.
func (s *QdrantStore) findPointByAgentID(ctx context.Context, agentID string) (*qdrant.RetrievedPoint, error) {
	points, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.collectionName,
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("id", agentID),
			},
		},
		Limit:       qdrant.PtrOf(uint32(1)),
		WithPayload: qdrant.NewWithPayload(true),
		WithVectors: qdrant.NewWithVectors(true),
	})
	if err != nil {
		return nil, fmt.Errorf("scroll: %w", err)
	}
	if len(points) == 0 {
		return nil, nil
	}
	return points[0], nil
}

// GetAgent retrieves an agent by ID from Qdrant.
func (s *QdrantStore) GetAgent(ctx context.Context, id string) (*RegisteredAgent, error) {
	point, err := s.findPointByAgentID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find agent: %w", err)
	}
	if point == nil {
		return nil, ErrNotFound
	}

	agent, err := payloadToAgent(id, point.Payload)
	if err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if point.Vectors != nil {
		if vec := point.Vectors.GetVector(); vec != nil {
			if dense := vec.GetDense(); dense != nil {
				agent.Embedding = dense.GetData()
			}
		}
	}

	return agent, nil
}

// ListAgents returns agents matching the filter criteria.
func (s *QdrantStore) ListAgents(ctx context.Context, filter AgentFilter) (*AgentListResult, error) {
	qdrantFilter := buildFilter(filter)

	// Scroll through all matching results
	points, err := s.scrollAll(ctx, qdrantFilter)
	if err != nil {
		return nil, fmt.Errorf("scroll points: %w", err)
	}

	// Convert to agents
	agents := make([]*RegisteredAgent, 0, len(points))
	for _, point := range points {
		id := point.Payload["id"].GetStringValue()
		agent, err := payloadToAgent(id, point.Payload)
		if err != nil {
			return nil, fmt.Errorf("parse payload for %s: %w", id, err)
		}
		agents = append(agents, agent)
	}

	// Sort by CreatedAt descending (matching memory.go behavior)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CreatedAt.After(agents[j].CreatedAt)
	})

	total := len(agents)

	// Apply pagination
	if filter.Limit == 0 {
		return &AgentListResult{
			Agents: []*RegisteredAgent{},
			Total:  total,
		}, nil
	}

	start := min(filter.Offset, len(agents))
	end := min(start+filter.Limit, len(agents))

	return &AgentListResult{
		Agents: agents[start:end],
		Total:  total,
	}, nil
}

// scrollAll fetches all matching points from the collection.
func (s *QdrantStore) scrollAll(ctx context.Context, filter *qdrant.Filter) ([]*qdrant.RetrievedPoint, error) {
	batchSize := uint32(100)
	var allPoints []*qdrant.RetrievedPoint
	var lastID *qdrant.PointId

	for {
		resp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: s.collectionName,
			Filter:         filter,
			Offset:         lastID,
			Limit:          qdrant.PtrOf(batchSize),
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			return nil, fmt.Errorf("scroll: %w", err)
		}

		allPoints = append(allPoints, resp...)

		if len(resp) < int(batchSize) {
			break
		}

		lastID = resp[len(resp)-1].Id
	}

	return allPoints, nil
}

// UpdateAgent updates an existing agent in Qdrant.
func (s *QdrantStore) UpdateAgent(ctx context.Context, agent *RegisteredAgent) error {
	// Find existing point
	point, err := s.findPointByAgentID(ctx, agent.ID)
	if err != nil {
		return fmt.Errorf("find agent: %w", err)
	}
	if point == nil {
		return ErrNotFound
	}

	payload, err := agentToPayload(agent)
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	// Reuse existing point ID
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      point.Id,
				Vectors: qdrant.NewVectorsDense(agent.Embedding),
				Payload: payload,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("upsert point: %w", err)
	}

	return nil
}

// DeleteAgent removes an agent from Qdrant.
func (s *QdrantStore) DeleteAgent(ctx context.Context, id string) error {
	// Find existing point
	point, err := s.findPointByAgentID(ctx, id)
	if err != nil {
		return fmt.Errorf("find agent: %w", err)
	}
	if point == nil {
		return ErrNotFound
	}

	_, err = s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrant.NewPointsSelector(point.Id),
	})
	if err != nil {
		return fmt.Errorf("delete point: %w", err)
	}

	return nil
}

// SearchAgents finds agents by vector similarity with optional filtering.
func (s *QdrantStore) SearchAgents(ctx context.Context, query []float32, limit int, filter AgentFilter) (*SearchResult, error) {
	qdrantFilter := buildFilter(filter)

	resp, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName,
		Query:          qdrant.NewQueryDense(query),
		Limit:          qdrant.PtrOf(uint64(limit)),
		Filter:         qdrantFilter,
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	agents := make([]ScoredAgent, 0, len(resp))
	for _, point := range resp {
		id := point.Payload["id"].GetStringValue()
		agent, err := payloadToAgent(id, point.Payload)
		if err != nil {
			return nil, fmt.Errorf("parse payload for %s: %w", id, err)
		}

		if point.Vectors != nil {
			if vec := point.Vectors.GetVector(); vec != nil {
				if dense := vec.GetDense(); dense != nil {
					agent.Embedding = dense.GetData()
				}
			}
		}

		agents = append(agents, ScoredAgent{
			Agent: agent,
			Score: point.Score,
		})
	}

	return &SearchResult{Agents: agents}, nil
}

// agentToPayload converts a RegisteredAgent to Qdrant payload.
func agentToPayload(agent *RegisteredAgent) (map[string]*qdrant.Value, error) {
	cardJSON, err := json.Marshal(agent.Card)
	if err != nil {
		return nil, fmt.Errorf("marshal agent card: %w", err)
	}

	skillIDs := make([]any, len(agent.Card.Skills))
	for i, skill := range agent.Card.Skills {
		skillIDs[i] = skill.ID
	}

	tags := make([]any, len(agent.Tags))
	for i, tag := range agent.Tags {
		tags[i] = tag
	}

	payload := map[string]any{
		"id":               agent.ID,
		"card":             string(cardJSON),
		"card_name":        agent.Card.Name,
		"card_description": agent.Card.Description,
		"tags":             tags,
		"skill_ids":        skillIDs,
		"created_at":       agent.CreatedAt.Unix(),
		"updated_at":       agent.UpdatedAt.Unix(),
	}

	return qdrant.NewValueMap(payload), nil
}

// payloadToAgent converts Qdrant payload to a RegisteredAgent.
func payloadToAgent(id string, payload map[string]*qdrant.Value) (*RegisteredAgent, error) {
	cardJSON := payload["card"].GetStringValue()
	var card a2a.AgentCard
	if err := json.Unmarshal([]byte(cardJSON), &card); err != nil {
		return nil, fmt.Errorf("unmarshal agent card: %w", err)
	}

	var tags []string
	if tagsValue := payload["tags"]; tagsValue != nil {
		if listVal := tagsValue.GetListValue(); listVal != nil {
			tags = make([]string, 0, len(listVal.GetValues()))
			for _, v := range listVal.GetValues() {
				tags = append(tags, v.GetStringValue())
			}
		}
	}

	createdAt := time.Unix(payload["created_at"].GetIntegerValue(), 0)
	updatedAt := time.Unix(payload["updated_at"].GetIntegerValue(), 0)

	return &RegisteredAgent{
		ID:        id,
		Card:      card,
		Tags:      tags,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// buildFilter converts AgentFilter to Qdrant Filter.
func buildFilter(filter AgentFilter) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// Tags filter: any tag matches
	if len(filter.Tags) > 0 {
		tagConditions := make([]*qdrant.Condition, len(filter.Tags))
		for i, tag := range filter.Tags {
			tagConditions[i] = qdrant.NewMatch("tags", tag)
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Filter{
				Filter: &qdrant.Filter{Should: tagConditions},
			},
		})
	}

	// Skills filter: any skill matches
	if len(filter.Skills) > 0 {
		skillConditions := make([]*qdrant.Condition, len(filter.Skills))
		for i, skill := range filter.Skills {
			skillConditions[i] = qdrant.NewMatch("skill_ids", skill)
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Filter{
				Filter: &qdrant.Filter{Should: skillConditions},
			},
		})
	}

	// Text query: search in name OR description
	if filter.Query != "" {
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Filter{
				Filter: &qdrant.Filter{
					Should: []*qdrant.Condition{
						qdrant.NewMatchText("card_name", filter.Query),
						qdrant.NewMatchText("card_description", filter.Query),
					},
				},
			},
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{Must: conditions}
}
