package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/a2a"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/store"
)

// AdminHandler handles admin API endpoints for agent management.
type AdminHandler struct {
	// registry is the service for agent operations.
	registry *registry.RegistryService
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(registry *registry.RegistryService) *AdminHandler {
	return &AdminHandler{registry: registry}
}

// RegisterRoutes registers admin routes on the given ServeMux.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/agents", h.handleList)
	mux.HandleFunc("POST /v1/admin/agents", h.handleCreate)
	mux.HandleFunc("GET /v1/admin/agents/{id}", h.handleGet)
	mux.HandleFunc("PUT /v1/admin/agents/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE /v1/admin/agents/{id}", h.handleDelete)
}

// RegisterAgentRequest is the JSON request for registering an agent.
type RegisterAgentRequest struct {
	// AgentID is the unique agent identifier.
	AgentID string `json:"agent_id"`
	// AgentCard is the A2A agent card.
	AgentCard a2a.AgentCard `json:"agent_card"`
	// Tags are classification tags.
	Tags []string `json:"tags"`
}

// UpdateAgentRequest is the JSON request for updating an agent.
type UpdateAgentRequest struct {
	// AgentCard is the updated A2A agent card.
	AgentCard a2a.AgentCard `json:"agent_card"`
	// Tags are the updated classification tags.
	Tags []string `json:"tags"`
}

// AgentRecordResponse is the JSON response for a single agent.
type AgentRecordResponse struct {
	// AgentID is the unique identifier.
	AgentID string `json:"agent_id"`
	// AgentCard is the A2A agent card.
	AgentCard a2a.AgentCard `json:"agent_card"`
	// Endpoint is the agent's URL.
	Endpoint string `json:"endpoint"`
	// Skills is the list of skill IDs.
	Skills []string `json:"skills"`
	// Tags are classification tags.
	Tags []string `json:"tags"`
	// RegisteredAt is the registration timestamp.
	RegisteredAt time.Time `json:"registered_at"`
	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
	// TODO: Add RegisteredBy field to track admin user who registered the agent.
}

// AgentListResponse is the JSON response for listing agents.
type AgentListResponse struct {
	// Agents is the list of agent records.
	Agents []AgentRecordResponse `json:"agents"`
	// Pagination contains pagination info.
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse contains pagination metadata.
type PaginationResponse struct {
	// Total is the total number of items.
	Total int `json:"total"`
	// Offset is the current offset.
	Offset int `json:"offset"`
	// Limit is items per page.
	Limit int `json:"limit"`
	// HasMore indicates if there are more items.
	HasMore bool `json:"has_more"`
}

// ErrorResponse is the JSON response for errors.
type ErrorResponse struct {
	// Code is the error code.
	Code string `json:"code"`
	// Message is the human-readable error message.
	Message string `json:"message"`
	// Details contains additional error details.
	Details map[string]any `json:"details,omitempty"`
}

func (h *AdminHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}

	agent, err := h.registry.Create(r.Context(), registry.CreateInput{
		ID:   req.AgentID,
		Card: req.AgentCard,
		Tags: req.Tags,
	})
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "AGENT_EXISTS",
				"agent with ID '"+req.AgentID+"' already exists")
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAgentResponse(agent))
}

func (h *AdminHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	agent, err := h.registry.Get(r.Context(), agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND",
				"agent with ID '"+agentID+"' not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toAgentResponse(agent))
}

func (h *AdminHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 20
	}

	var tags, skills []string
	if t := query.Get("tags"); t != "" {
		tags = strings.Split(t, ",")
	}
	if s := query.Get("skills"); s != "" {
		skills = strings.Split(s, ",")
	}

	result, err := h.registry.List(r.Context(), registry.ListInput{
		Offset: offset,
		Limit:  limit,
		Tags:   tags,
		Skills: skills,
		Query:  query.Get("q"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	agents := make([]AgentRecordResponse, len(result.Agents))
	for i, a := range result.Agents {
		agents[i] = toAgentResponse(a)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AgentListResponse{
		Agents: agents,
		Pagination: PaginationResponse{
			Total:   result.Total,
			Offset:  offset,
			Limit:   limit,
			HasMore: offset+len(agents) < result.Total,
		},
	})
}

func (h *AdminHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	var req UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body")
		return
	}

	agent, err := h.registry.Update(r.Context(), registry.UpdateInput{
		ID:   agentID,
		Card: req.AgentCard,
		Tags: req.Tags,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND",
				"agent with ID '"+agentID+"' not found")
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toAgentResponse(agent))
}

func (h *AdminHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	if err := h.registry.Delete(r.Context(), agentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND",
				"agent with ID '"+agentID+"' not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toAgentResponse(agent *store.RegisteredAgent) AgentRecordResponse {
	skills := make([]string, len(agent.Card.Skills))
	for i, s := range agent.Card.Skills {
		skills[i] = s.ID
	}

	tags := agent.Tags
	if tags == nil {
		tags = []string{}
	}

	return AgentRecordResponse{
		AgentID:      agent.ID,
		AgentCard:    agent.Card,
		Endpoint:     agent.Card.URL,
		Skills:       skills,
		Tags:         tags,
		RegisteredAt: agent.CreatedAt,
		UpdatedAt:    agent.UpdatedAt,
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}
