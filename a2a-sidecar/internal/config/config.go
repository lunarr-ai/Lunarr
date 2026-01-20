package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
)

const (
	DefaultPort           = 8080
	DefaultTimeoutSeconds = 300
	DefaultMaxConcurrent  = 100
	DefaultLogLevel       = slog.LevelInfo
	DefaultAgentType      = "a2a"
)

var (
	ErrMissingAgentCard = errors.New("AGENTCARD environment variable is required")
	ErrInvalidAgentCard = errors.New("invalid AGENTCARD JSON")
	ErrInvalidCardURL   = errors.New("invalid card.url")
	ErrMissingCardName  = errors.New("card.name is required")
	ErrMissingCardURL   = errors.New("card.url is required")
	ErrInvalidAgentType = errors.New("invalid AGENT_TYPE")
)

type Config struct {
	// Port is the HTTP server port.
	Port int
	// TimeoutSeconds is the request timeout in seconds.
	TimeoutSeconds int
	// MaxConcurrent is the maximum concurrent requests.
	MaxConcurrent int
	// LogLevel is the minimum log level for logging.
	LogLevel slog.Level
	// Card is the A2A agent card.
	Card a2a.AgentCard
	// EndpointPath is the path extracted from card.url.
	EndpointPath string
	// BackendURL is the derived backend URL.
	BackendURL string
	// AgentType is the backend agent type (a2a, adk).
	AgentType string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:           DefaultPort,
		TimeoutSeconds: DefaultTimeoutSeconds,
		MaxConcurrent:  DefaultMaxConcurrent,
		LogLevel:       DefaultLogLevel,
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		cfg.Port = port
	}

	if timeoutStr := os.Getenv("TIMEOUT_SECONDS"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TIMEOUT_SECONDS: %w", err)
		}
		cfg.TimeoutSeconds = timeout
	}

	if maxConcStr := os.Getenv("MAX_CONCURRENT"); maxConcStr != "" {
		maxConc, err := strconv.Atoi(maxConcStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_CONCURRENT: %w", err)
		}
		cfg.MaxConcurrent = maxConc
	}

	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(logLevelStr)); err != nil {
			return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
		cfg.LogLevel = level
	}

	cfg.AgentType = os.Getenv("AGENT_TYPE")
	if cfg.AgentType == "" {
		cfg.AgentType = DefaultAgentType
	}
	// Validate agent type
	if cfg.AgentType != "a2a" && cfg.AgentType != "adk" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAgentType, cfg.AgentType)
	}

	agentCardJSON := os.Getenv("AGENTCARD")
	if agentCardJSON == "" {
		return nil, ErrMissingAgentCard
	}

	if err := json.Unmarshal([]byte(agentCardJSON), &cfg.Card); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAgentCard, err)
	}

	// Validate required card fields
	if cfg.Card.Name == "" {
		return nil, ErrMissingCardName
	}
	if cfg.Card.URL == "" {
		return nil, ErrMissingCardURL
	}

	cardURL, err := url.Parse(cfg.Card.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCardURL, err)
	}
	cfg.EndpointPath = cardURL.Path

	// Derive backend URL: http://{card.name}:{port}/{path}
	sanitizedName := sanitizeName(cfg.Card.Name)
	cfg.BackendURL = fmt.Sprintf("http://%s:%d%s", sanitizedName, cfg.Port, cfg.EndpointPath)

	return cfg, nil
}

// sanitizeName converts agent name to a valid hostname.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
