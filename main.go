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
)

func main() {
	cfg, err := ParseConfig()
	if err != nil {
		slog.Error("failed to parse config", "error", err)
		os.Exit(1)
	}

	handler, err := newRedirectHandler(cfg.Destination, cfg.StatusCode)
	if err != nil {
		slog.Error("failed to create handler", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(handler),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in background
	go func() {
		slog.Info("redirector starting",
			"addr", addr,
			"destination", cfg.Destination,
			"status", cfg.StatusCode,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}
