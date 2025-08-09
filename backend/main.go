package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	AIFilter    bool `json:"ai_filter"`
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
	startedAt := time.Now()
	rid := fmt.Sprintf("%d", time.Now().UnixNano())
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("rid=%s decode_error err=%v", rid, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Settings.Queries == 0 {
		req.Settings.Queries = cfg.DefaultQueryCount
	}

	log.Printf("rid=%s search_start prompt=%q settings={queries:%d content_mode:%t ai_filter:%t}", rid, truncateStr(req.Prompt, 200), req.Settings.Queries, req.Settings.ContentMode, req.Settings.AIFilter)

	// Step 1: Generate queries via OpenRouter
	queries, err := generateQueriesWithOpenRouter(req.Prompt, req.Settings.Queries, cfg.OpenRouterAPIKey)
	if err != nil {
		log.Printf("rid=%s queries_error err=%v", rid, err)
		http.Error(w, "failed to generate queries: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("rid=%s queries_ok count=%d sample=%v", rid, len(queries), sampleStrings(queries, 3))

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
				log.Printf("rid=%s searx_error query=%q err=%v", rid, q, err)
				return err
			}
			mu.Lock()
			results = append(results, res...)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Printf("rid=%s searx_group_error err=%v", rid, err)
		http.Error(w, "search error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// deduplicate & rank
	ranked := deduplicateAndRank(results)
	log.Printf("rid=%s rank_done in_count=%d out_count=%d", rid, len(results), len(ranked))

	// If content mode requested, fetch page content and evaluate relevance for each item individually
	if req.Settings.ContentMode {
		type contentEval struct {
			idx     int
			content string
			keep    bool
			err     error
		}

		resultsCh := make(chan contentEval, len(ranked))

		var eg2 errgroup.Group
		for i := range ranked {
			i := i
			eg2.Go(func() error {
				ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
				defer cancel()
				content, err := fetchPageContent(ctx, ranked[i].URL)
				if err != nil {
					log.Printf("rid=%s content_fetch_error url=%q err=%v", rid, ranked[i].URL, err)
					resultsCh <- contentEval{idx: i, err: err}
					return nil
				}
				// Evaluate relevance per-item to avoid huge prompts
				relevant, relErr := isContentRelevantToPrompt(req.Prompt, ranked[i].Title, ranked[i].URL, content, cfg.OpenRouterAPIKey)
				if relErr != nil {
					log.Printf("rid=%s content_relevance_error url=%q err=%v", rid, ranked[i].URL, relErr)
				}
				resultsCh <- contentEval{idx: i, content: content, keep: relErr == nil && relevant, err: relErr}
				return nil
			})
		}
		_ = eg2.Wait()
		close(resultsCh)

		// Build filtered list: keep items that were judged relevant; if fetch failed, keep original
		keepMap := make(map[int]bool, len(ranked))
		for r := range resultsCh {
			if r.err != nil {
				// fetch or relevance error -> keep original item to degrade gracefully
				keepMap[r.idx] = true
				continue
			}
			keepMap[r.idx] = r.keep
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
		// Regular mode: optionally apply AI relevance filtering on snippets produced by Searx
		if req.Settings.AIFilter {
			log.Printf("rid=%s ai_filter_start items=%d", rid, len(ranked))
			filtered, err := filterByAIRelevance(req.Prompt, ranked, cfg.OpenRouterAPIKey)
			if err == nil {
				ranked = filtered
				log.Printf("rid=%s ai_filter_ok out_items=%d", rid, len(ranked))
			} else {
				log.Printf("rid=%s ai_filter_error err=%v", rid, err)
			}
		}
	}

	resp := SearchResponse{
		Queries: queries,
		Results: ranked,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("rid=%s search_done duration_ms=%d results=%d", rid, time.Since(startedAt).Milliseconds(), len(ranked))
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
