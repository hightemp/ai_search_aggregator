package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	cfg := loadConfig()
	logger := NewLogger()

	// Создаем главный роутер
	mainRouter := http.NewServeMux()

	// Отдельный обработчик для WebSocket без middleware
	mainRouter.HandleFunc("/api/ws/search", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketSearch(w, r, cfg, logger)
	})

	// Chi роутер для остальных эндпоинтов с middleware
	r := chi.NewRouter()
	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware(logger))

	// CORS для обычных HTTP эндпоинтов
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Монтируем chi роутер для всех путей кроме WebSocket
	mainRouter.Handle("/", r)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mainRouter,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server exited")
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func sampleStrings(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return in[:n]
}
