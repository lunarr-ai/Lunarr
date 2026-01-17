package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/lunarr-ai/lunarr/agent-broker/internal/agent"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/config"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/handler"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/registry"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/server"
	"github.com/lunarr-ai/lunarr/agent-broker/internal/store"
	"github.com/lunarr-ai/lunarr/agent-broker/pkg/embedding"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg := config.Load()
	logger := setupLogger(cfg.LogLevel)

	logger.Info("starting agent-broker",
		"port", cfg.Port,
		"log_level", cfg.LogLevel.String(),
		"qdrant_host", cfg.QdrantHost,
		"qdrant_port", cfg.QdrantPort,
		"embedding_url", cfg.EmbeddingURL,
		"embedding_dim", cfg.EmbeddingDim,
	)

	ctx := context.Background()

	// Create embedder with configured dimension
	embedder := embedding.NewClient(cfg.EmbeddingURL, cfg.EmbeddingDim)

	// Create Qdrant store with configured dimension
	qdrantStore, err := store.NewQdrantStore(ctx,
		store.WithHost(cfg.QdrantHost),
		store.WithPort(cfg.QdrantPort),
		store.WithAPIKey(cfg.QdrantAPIKey),
		store.WithTLS(cfg.QdrantUseTLS),
		store.WithVectorDimension(uint64(cfg.EmbeddingDim)),
	)
	if err != nil {
		logger.Error("failed to connect to qdrant", "error", err)
		return err
	}
	defer func() {
		if err := qdrantStore.Close(); err != nil {
			logger.Error("failed to close qdrant connection", "error", err)
		}
	}()
	logger.Info("connected to qdrant")

	registryService := registry.NewRegistryService(qdrantStore, registry.WithEmbedder(embedder))

	brokerAgent, err := agent.NewBrokerAgent(ctx, registryService,
		agent.WithGeminiAPIKey(cfg.GeminiAPIKey),
		agent.WithGeminiModel(cfg.GeminiModel),
	)
	if err != nil {
		logger.Error("failed to create broker agent", "error", err)
		return err
	}

	sessionService := agent.NewSessionService()

	mux := http.NewServeMux()

	handler.NewBrokerHandler(brokerAgent, sessionService).RegisterRoutes(mux)
	handler.NewHealthHandler(qdrantStore).RegisterRoutes(mux)
	handler.NewAdminHandler(registryService).RegisterRoutes(mux)
	handler.NewAgentsHandler(registryService).RegisterRoutes(mux)

	srv := server.New(mux,
		server.WithPort(cfg.Port),
		server.WithLogger(logger),
	)

	if err := srv.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		return err
	}

	return nil
}

func setupLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
