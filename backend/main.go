package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"golang.org/x/sync/errgroup"
)

type SearchRequest struct {
	Prompt   string   `json:"prompt"`
	Settings Settings `json:"settings"`
}

type Settings struct {
	Queries     int      `json:"queries"`
	ContentMode bool     `json:"content_mode"`
	Engines     []string `json:"engines"`
}

type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type SearchResponse struct {
	Queries []string       `json:"queries"`
	Results []SearchResult `json:"results"`
}

func main() {
	cfg := loadConfig()
	logger := NewLogger()

	r := chi.NewRouter()

	// Middleware
	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware(logger))

	// Very permissive CORS for the MVP
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

	// Handler closure captures cfg and logger
	r.Post("/api/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, cfg, logger)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("starting server", "port", cfg.Port)
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

func handleSearch(w http.ResponseWriter, r *http.Request, cfg AppConfig, logger *Logger) {
	startedAt := time.Now()
	type requestIDKey string
	const ridKey requestIDKey = "request_id"
	rid, _ := r.Context().Value(ridKey).(string)
	if rid == "" {
		rid = generateRequestID()
	}
	reqLogger := logger.WithRequestID(rid)

	// Устанавливаем общий тайм-аут для запроса
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqLogger.Error("failed to decode request", err)
		ErrorResponse(w, WrapError(ErrInvalidRequest, err))
		return
	}

	// Санитизация входных данных
	SanitizeSearchRequest(&req)

	// Валидация входных данных
	if validationErrors := ValidateSearchRequest(&req); len(validationErrors) > 0 {
		reqLogger.Error("validation failed", validationErrors)
		appErr := &AppError{
			Code:    "VALIDATION_FAILED",
			Message: "Request validation failed",
			Details: validationErrors.Error(),
			Status:  http.StatusBadRequest,
		}
		ErrorResponse(w, appErr)
		return
	}

	if req.Settings.Queries == 0 {
		req.Settings.Queries = cfg.DefaultQueryCount
	}

	reqLogger.Info("search_start",
		"prompt", truncateStr(req.Prompt, 200),
		"queries", req.Settings.Queries,
		"content_mode", req.Settings.ContentMode,
		"engines", req.Settings.Engines,
	)

	// Step 1: Generate queries via OpenRouter
	queries, err := generateQueriesWithOpenRouter(ctx, req.Prompt, req.Settings.Queries, cfg.OpenRouterAPIKey)
	if err != nil {
		reqLogger.Error("failed to generate queries", err)
		ErrorResponse(w, WrapError(ErrQueryGeneration, err))
		return
	}
	reqLogger.Info("queries_generated", "count", len(queries), "sample", sampleStrings(queries, 3))

	// Step 2: Execute Searx searches concurrently
	var (
		eg      errgroup.Group
		mu      sync.Mutex
		results []SearchResult
	)

	// Ограничиваем количество одновременных поисковых запросов
	eg.SetLimit(5)

	for _, q := range queries {
		q := q // capture loop var
		eg.Go(func() error {
			queryCtx, queryCancel := context.WithTimeout(ctx, 30*time.Second)
			defer queryCancel()

			res, err := searchSearx(queryCtx, cfg.SearxURL, q, req.Settings.Engines)
			if err != nil {
				reqLogger.Error("searx search failed", err, "query", q)
				return err
			}
			mu.Lock()
			results = append(results, res...)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		reqLogger.Error("searx search group failed", err)
		ErrorResponse(w, WrapError(ErrSearchFailed, err))
		return
	}

	// deduplicate & rank
	ranked := deduplicateAndRank(results)
	reqLogger.Info("deduplication_completed", "input_count", len(results), "output_count", len(ranked))

	// If content mode requested, fetch page content and evaluate relevance for each item individually
	if req.Settings.ContentMode {
		type contentEval struct {
			idx         int
			content     string
			keep        bool
			fetchFailed bool
			err         error
		}

		resultsCh := make(chan contentEval, len(ranked))

		var eg2 errgroup.Group
		// Ограничиваем количество одновременных запросов контента
		eg2.SetLimit(3)

		for i := range ranked {
			i := i
			eg2.Go(func() error {
				contentCtx, contentCancel := context.WithTimeout(ctx, 20*time.Second)
				defer contentCancel()

				content, err := fetchPageContent(contentCtx, ranked[i].URL)
				if err != nil {
					reqLogger.Error("content fetch failed", err, "url", ranked[i].URL)
					resultsCh <- contentEval{idx: i, fetchFailed: true, err: err}
					return nil
				}
				// Evaluate relevance per-item to avoid huge prompts
				relevant, relErr := isContentRelevantToPrompt(contentCtx, req.Prompt, ranked[i].Title, ranked[i].URL, content, cfg.OpenRouterAPIKey)
				if relErr != nil {
					reqLogger.Error("content relevance evaluation failed", relErr, "url", ranked[i].URL)
				}
				resultsCh <- contentEval{idx: i, content: content, keep: relErr == nil && relevant, err: relErr}
				return nil
			})
		}
		_ = eg2.Wait()
		close(resultsCh)

		// Build filtered list: keep items that were judged relevant; if fetch failed, drop item
		keepMap := make(map[int]bool, len(ranked))
		for r := range resultsCh {
			if r.fetchFailed {
				keepMap[r.idx] = false
			} else if r.err != nil {
				// relevance error -> keep original item (graceful degrade)
				keepMap[r.idx] = true
			} else {
				keepMap[r.idx] = r.keep
			}
		}

		filtered := make([]SearchResult, 0, len(ranked))
		for i, it := range ranked {
			if keep, ok := keepMap[i]; ok {
				if keep {
					filtered = append(filtered, it)
				}
			} else {
				// No evaluation result (should not happen) -> keep
				filtered = append(filtered, it)
			}
		}
		ranked = filtered
	} else {
		reqLogger.Info("ai_filter_start", "items", len(ranked))
		filtered, err := filterByAIRelevance(ctx, req.Prompt, ranked, cfg.OpenRouterAPIKey)
		if err == nil {
			ranked = filtered
			reqLogger.Info("ai_filter_completed", "output_items", len(ranked))
		} else {
			reqLogger.Error("ai_filter_failed", err)
		}
	}

	resp := SearchResponse{
		Queries: queries,
		Results: ranked,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		reqLogger.Error("failed to encode response", err)
		ErrorResponse(w, &AppError{
			Code:    "RESPONSE_ENCODING_FAILED",
			Message: "Failed to encode response",
			Status:  http.StatusInternalServerError,
		})
		return
	}

	reqLogger.WithDuration(startedAt).Info("search_completed", "results", len(ranked))
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
