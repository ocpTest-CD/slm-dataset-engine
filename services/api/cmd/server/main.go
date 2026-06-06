package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/api"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/config"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/repository"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/service"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/storage"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.ApplyMigrations(ctx, cfg.MigrationsDir); err != nil {
		logger.Error("apply migrations", "error", err)
		os.Exit(1)
	}

	localStorage := storage.NewLocal(cfg.ArtifactsDir)
	app := service.New(store, localStorage, logger)
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.NewRouter(app, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api server started", "addr", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api shutdown", "error", err)
	}
}
