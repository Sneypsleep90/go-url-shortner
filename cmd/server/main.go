package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sneypsleep90/go-url-shortner/internal/config"
	"github.com/Sneypsleep90/go-url-shortner/internal/handler"
	"github.com/Sneypsleep90/go-url-shortner/internal/service"
	"github.com/Sneypsleep90/go-url-shortner/internal/storage/memory"
	"github.com/Sneypsleep90/go-url-shortner/internal/storage/postgres"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var repo service.LinkRepo
	switch cfg.StorageType {
	case config.StorageMemory:
		repo = memory.New()
	case config.StoragePostgres:
		pool, err := pgxpool.New(ctx, cfg.DataBaseUrl)
		if err != nil {
			return fmt.Errorf("create postgres pool: %w", err)
		}
		defer pool.Close()

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			return fmt.Errorf("connect to postgres: %w", err)
		}

		repo = postgres.NewPgStorage(pool)
	default:
		return fmt.Errorf("unsupported storage type: %q", cfg.StorageType)
	}
	logger.Info("storage initialized", slog.String("storage_type", cfg.StorageType))

	svc := service.NewService(repo)
	h := handler.NewHandler(svc)

	router := gin.New()
	router.Use(handler.RequestLogger(logger), gin.Recovery())
	h.RegisterRoutes(router)

	return serve(
		ctx,
		cfg.HTTPAddr,
		router,
		logger,
	)
}

func serve(ctx context.Context, addr string, router http.Handler, logger *slog.Logger) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	defer server.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	logger.Info("starting HTTP server", slog.String("address", addr))

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}
	logger.Info("stopping HTTP server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	logger.Info("HTTP server stopped")
	return nil
}
