package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
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
	Queries     int  `json:"queries"`
	ContentMode bool `json:"content_mode"`
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

	r := chi.NewRouter()

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

	// Handler closure captures cfg
	r.Post("/api/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, cfg)
	})

	log.Printf("starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request, cfg AppConfig) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Settings.Queries == 0 {
		req.Settings.Queries = cfg.DefaultQueryCount
	}

	// Step 1: Generate queries via OpenRouter
	queries, err := generateQueriesWithOpenRouter(req.Prompt, req.Settings.Queries, cfg.OpenRouterAPIKey)
	if err != nil {
		http.Error(w, "failed to generate queries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 2: Execute Searx searches concurrently
	var (
		eg      errgroup.Group
		mu      sync.Mutex
		results []SearchResult
	)

	for _, q := range queries {
		q := q // capture loop var
		eg.Go(func() error {
			res, err := searchSearx(cfg.SearxURL, q)
			if err != nil {
				return err
			}
			mu.Lock()
			results = append(results, res...)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// deduplicate & rank
	ranked := deduplicateAndRank(results)

	// If content mode requested, fetch page content concurrently and replace snippet
	if req.Settings.ContentMode {
		var eg2 errgroup.Group
		for i := range ranked {
			i := i
			eg2.Go(func() error {
				ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
				defer cancel()
				content, err := fetchPageContent(ctx, ranked[i].URL)
				if err == nil && content != "" {
					ranked[i].Snippet = content
				}
				return nil
			})
		}
		_ = eg2.Wait()
	}

	resp := SearchResponse{
		Queries: queries,
		Results: ranked,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
