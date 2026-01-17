package handler

import (
	"context"
	"iter"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
)

// Skill IDs for the broker's A2A skills.
const (
	skillDiscover  = "discover"
	skillRoute     = "route"
	skillBroadcast = "broadcast"
)

// Broker metadata.
const (
	brokerName         = "Lunarr Agent Broker"
	brokerDescription  = "A2A-compliant meta-agent for agent discovery, routing, and broadcast"
	brokerVersion      = "1.0.0"
	a2aProtocolVersion = "0.2.5"
)

// BrokerHandler implements a2asrv.RequestHandler for the broker's skills.
type BrokerHandler struct {
	// registry is the agent registry service.
	registry *registry.RegistryService
	// brokerURL is the URL where this broker is accessible.
	brokerURL string
}

// NewBrokerHandler creates a new BrokerHandler.
func NewBrokerHandler(registry *registry.RegistryService, brokerURL string) *BrokerHandler {
	return &BrokerHandler{registry: registry, brokerURL: brokerURL}
}

// RegisterRoutes registers broker A2A routes on the given ServeMux.
func (h *BrokerHandler) RegisterRoutes(mux *http.ServeMux) {
	card := brokerCard(h.brokerURL)
	mux.Handle("POST /", a2asrv.NewJSONRPCHandler(h))
	mux.Handle("GET /.well-known/agent-card.json", a2asrv.NewStaticAgentCardHandler(card))
}

// OnSendMessage handles message/send - dispatches to discover/route/broadcast skills.
func (h *BrokerHandler) OnSendMessage(ctx context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	skill := extractSkill(params.Message)

	switch skill {
	case skillDiscover:
		return h.handleDiscover(ctx, params)
	case skillRoute:
		return h.handleRoute(ctx, params)
	case skillBroadcast:
		return h.handleBroadcast(ctx, params)
	default:
		return nil, a2a.ErrInvalidParams
	}
}

// OnSendMessageStream handles message/stream (streaming support in Phase 9).
func (h *BrokerHandler) OnSendMessageStream(_ context.Context, _ *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(nil, a2a.ErrUnsupportedOperation)
	}
}

// OnGetTask handles tasks/get.
func (h *BrokerHandler) OnGetTask(_ context.Context, _ *a2a.TaskQueryParams) (*a2a.Task, error) {
	return nil, a2a.ErrUnsupportedOperation
}

// OnCancelTask handles tasks/cancel.
func (h *BrokerHandler) OnCancelTask(_ context.Context, _ *a2a.TaskIDParams) (*a2a.Task, error) {
	return nil, a2a.ErrUnsupportedOperation
}

// OnResubscribeToTask handles tasks/resubscribe.
func (h *BrokerHandler) OnResubscribeToTask(_ context.Context, _ *a2a.TaskIDParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(nil, a2a.ErrUnsupportedOperation)
	}
}

// OnGetTaskPushConfig handles tasks/pushNotificationConfig/get.
func (h *BrokerHandler) OnGetTaskPushConfig(_ context.Context, _ *a2a.GetTaskPushConfigParams) (*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrPushNotificationNotSupported
}

// OnListTaskPushConfig handles tasks/pushNotificationConfig/list.
func (h *BrokerHandler) OnListTaskPushConfig(_ context.Context, _ *a2a.ListTaskPushConfigParams) ([]*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrPushNotificationNotSupported
}

// OnSetTaskPushConfig handles tasks/pushNotificationConfig/set.
func (h *BrokerHandler) OnSetTaskPushConfig(_ context.Context, _ *a2a.TaskPushConfig) (*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrPushNotificationNotSupported
}

// OnDeleteTaskPushConfig handles tasks/pushNotificationConfig/delete.
func (h *BrokerHandler) OnDeleteTaskPushConfig(_ context.Context, _ *a2a.DeleteTaskPushConfigParams) error {
	return a2a.ErrPushNotificationNotSupported
}

// OnGetExtendedAgentCard handles agent/getAuthenticatedExtendedCard.
func (h *BrokerHandler) OnGetExtendedAgentCard(_ context.Context) (*a2a.AgentCard, error) {
	return nil, a2a.ErrAuthenticatedExtendedCardNotConfigured
}

// extractSkill extracts skill ID from message DataPart.
func extractSkill(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if dp, ok := part.(*a2a.DataPart); ok {
			if skill, ok := dp.Data["skill"].(string); ok {
				return skill
			}
		}
	}
	return ""
}

func (h *BrokerHandler) handleDiscover(ctx context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	req, err := parseDiscoverRequest(params.Message)
	if err != nil {
		return nil, a2a.ErrInvalidParams
	}

	result, err := h.registry.Discover(ctx, registry.DiscoverInput{
		Query:  req.query,
		Limit:  req.limit,
		Tags:   req.tags,
		Skills: req.skills,
	})
	if err != nil {
		return nil, a2a.ErrInternalError
	}

	agents := make([]map[string]any, 0, len(result.Agents))
	for _, scored := range result.Agents {
		agents = append(agents, map[string]any{
			"card":  scored.Agent.Card,
			"score": scored.Score,
		})
	}

	return &a2a.Message{
		Role: a2a.MessageRoleAgent,
		Parts: []a2a.Part{
			&a2a.DataPart{
				Data: map[string]any{
					"agents": agents,
					"total":  len(agents),
				},
			},
		},
	}, nil
}

type discoverRequest struct {
	query  string
	limit  int
	tags   []string
	skills []string
}

func parseDiscoverRequest(msg *a2a.Message) (*discoverRequest, error) {
	if msg == nil {
		return nil, a2a.ErrInvalidParams
	}

	for _, part := range msg.Parts {
		dp, ok := part.(*a2a.DataPart)
		if !ok {
			continue
		}

		query, _ := dp.Data["query"].(string)
		if query == "" {
			return nil, a2a.ErrInvalidParams
		}

		req := &discoverRequest{query: query}

		if limit, ok := dp.Data["limit"].(float64); ok {
			req.limit = int(limit)
		}

		if tags, ok := dp.Data["tags"].([]any); ok {
			for _, t := range tags {
				if s, ok := t.(string); ok {
					req.tags = append(req.tags, s)
				}
			}
		}

		if skills, ok := dp.Data["skills"].([]any); ok {
			for _, s := range skills {
				if str, ok := s.(string); ok {
					req.skills = append(req.skills, str)
				}
			}
		}

		return req, nil
	}

	return nil, a2a.ErrInvalidParams
}

func (h *BrokerHandler) handleRoute(_ context.Context, _ *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (h *BrokerHandler) handleBroadcast(_ context.Context, _ *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	return nil, a2a.ErrUnsupportedOperation
}

// brokerCard returns the broker's A2A agent card.
func brokerCard(url string) *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:            brokerName,
		Description:     brokerDescription,
		URL:             url,
		Version:         brokerVersion,
		ProtocolVersion: a2aProtocolVersion,
		Skills: []a2a.AgentSkill{
			{
				ID:          skillDiscover,
				Name:        "Discover Agents",
				Description: "Find agents by query, tags, or skills",
			},
			{
				ID:          skillRoute,
				Name:        "Route Request",
				Description: "Forward task to best-matching agent",
			},
			{
				ID:          skillBroadcast,
				Name:        "Broadcast Request",
				Description: "Send task to multiple agents",
			},
		},
	}
}
