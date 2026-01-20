package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/lunarr-ai/lunarr/a2a-sidecar/internal/config"
	"github.com/lunarr-ai/lunarr/a2a-sidecar/internal/handler"
	"github.com/lunarr-ai/lunarr/a2a-sidecar/internal/proxy"
	"github.com/lunarr-ai/lunarr/a2a-sidecar/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("failed to run", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	logger.Info("starting a2a-sidecar",
		"port", cfg.Port,
		"agent_name", cfg.Card.Name,
		"agent_type", cfg.AgentType,
		"endpoint_path", cfg.EndpointPath,
		"backend_url", cfg.BackendURL,
		"timeout_seconds", cfg.TimeoutSeconds,
		"max_concurrent", cfg.MaxConcurrent,
	)

	// Create translator based on agent type
	var translator proxy.Translator
	switch cfg.AgentType {
	case "a2a":
		translator = proxy.NewA2ATranslator(cfg.BackendURL)
	case "adk":
		// Future: translator = proxy.NewADKTranslator(cfg.BackendURL)
		translator = proxy.NewA2ATranslator(cfg.BackendURL)
	}

	executor := proxy.NewProxyExecutor(translator,
		proxy.WithTimeout(time.Duration(cfg.TimeoutSeconds)*time.Second),
		proxy.WithLogger(logger),
	)

	requestHandler := a2asrv.NewHandler(executor)

	mux := http.NewServeMux()

	handler.NewHealthHandler().RegisterRoutes(mux)
	mux.Handle("GET /.well-known/agent-card.json", a2asrv.NewStaticAgentCardHandler(&cfg.Card))
	mux.Handle("POST "+cfg.EndpointPath, a2asrv.NewJSONRPCHandler(requestHandler))

	srv := server.New(mux,
		server.WithPort(cfg.Port),
		server.WithLogger(logger),
	)

	return srv.Run(context.Background())
}
