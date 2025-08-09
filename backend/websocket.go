package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// Базовые типы для поиска
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене здесь должна быть более строгая проверка
		return true
	},
}

// Типы сообщений WebSocket
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type WSSearchRequest struct {
	Prompt   string   `json:"prompt"`
	Settings Settings `json:"settings"`
}

type WSSearchStatus struct {
	Stage     string `json:"stage"`
	Progress  int    `json:"progress"`
	Total     int    `json:"total"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type WSSearchResult struct {
	Queries []string       `json:"queries"`
	Results []SearchResult `json:"results"`
	Elapsed int64          `json:"elapsed_ms"`
}

type WSError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
}

func handleWebSocketSearch(w http.ResponseWriter, r *http.Request, cfg AppConfig, logger *Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("failed to upgrade connection", err)
		return
	}
	defer conn.Close()

	// Создаем контекст с таймаутом для всего поиска
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	reqLogger := logger.WithRequestID(generateRequestID())
	reqLogger.Info("websocket connection established")

	// Обработка входящих сообщений
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				reqLogger.Error("websocket read error", err)
			}
			break
		}

		if msg.Type == "search" {
			go handleSearchMessage(ctx, conn, msg, cfg, reqLogger.logger)
		}
	}
}

func handleSearchMessage(ctx context.Context, conn *websocket.Conn, msg WSMessage, cfg AppConfig, logger *Logger) {
	startTime := time.Now()

	// Парсим поисковый запрос
	reqData, err := json.Marshal(msg.Data)
	if err != nil {
		sendError(conn, "INVALID_REQUEST", "Failed to parse request", err.Error())
		return
	}

	var req WSSearchRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		sendError(conn, "INVALID_REQUEST", "Failed to decode request", err.Error())
		return
	}

	// Санитизация и валидация
	searchReq := SearchRequest{
		Prompt:   req.Prompt,
		Settings: req.Settings,
	}
	SanitizeSearchRequest(&searchReq)

	if validationErrors := ValidateSearchRequest(&searchReq); len(validationErrors) > 0 {
		sendError(conn, "VALIDATION_FAILED", "Request validation failed", validationErrors.Error())
		return
	}

	if searchReq.Settings.Queries == 0 {
		searchReq.Settings.Queries = cfg.DefaultQueryCount
	}

	logger.Info("websocket search started",
		"prompt", truncateStr(searchReq.Prompt, 200),
		"queries", searchReq.Settings.Queries,
		"content_mode", searchReq.Settings.ContentMode,
	)

	// Отправляем статус начала поиска
	sendStatus(conn, "generating_queries", 0, 1, "Генерация поисковых запросов...")

	// Шаг 1: Генерация запросов
	queries, err := generateQueriesWithOpenRouter(ctx, searchReq.Prompt, searchReq.Settings.Queries, cfg.OpenRouterAPIKey)
	if err != nil {
		sendError(conn, "QUERY_GENERATION_FAILED", "Failed to generate queries", err.Error())
		return
	}

	logger.Info("queries generated", "count", len(queries))
	sendStatus(conn, "searching", 0, len(queries), "Выполнение поисковых запросов...")

	// Шаг 2: Выполнение поисков
	var (
		eg        errgroup.Group
		mu        sync.Mutex
		results   []SearchResult
		completed int
	)

	eg.SetLimit(5) // Ограничиваем количество одновременных запросов

	for _, query := range queries {
		query := query
		eg.Go(func() error {
			queryCtx, queryCancel := context.WithTimeout(ctx, 30*time.Second)
			defer queryCancel()

			res, err := searchSearx(queryCtx, cfg.SearxURL, query, searchReq.Settings.Engines)
			if err != nil {
				logger.Error("searx search failed", "error", err, "query", query)
				return err
			}

			mu.Lock()
			results = append(results, res...)
			completed++
			mu.Unlock()

			// Отправляем обновление прогресса
			sendStatus(conn, "searching", completed, len(queries),
				"Выполнено запросов: %d/%d", completed, len(queries))

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		logger.Error("searx search group failed", "error", err)
		sendError(conn, "SEARCH_FAILED", "Search failed", err.Error())
		return
	}

	// Дедупликация и ранжирование
	sendStatus(conn, "processing", 0, 1, "Обработка результатов...")
	ranked := deduplicateAndRank(results)
	logger.Info("deduplication completed", "input_count", len(results), "output_count", len(ranked))

	// Фильтрация по релевантности
	if searchReq.Settings.ContentMode {
		sendStatus(conn, "analyzing_content", 0, len(ranked), "Анализ содержимого страниц...")
		ranked = analyzeContentWithProgress(ctx, conn, searchReq.Prompt, ranked, cfg, logger)
	} else {
		sendStatus(conn, "ai_filtering", 0, 1, "ИИ-фильтрация результатов...")
		filtered, err := filterByAIRelevance(ctx, searchReq.Prompt, ranked, cfg.OpenRouterAPIKey)
		if err == nil {
			ranked = filtered
			logger.Info("ai filter completed", "output_items", len(ranked))
		} else {
			logger.Error("ai filter failed", "error", err)
		}
	}

	elapsed := time.Since(startTime).Milliseconds()

	// Отправляем финальные результаты
	response := WSSearchResult{
		Queries: queries,
		Results: ranked,
		Elapsed: elapsed,
	}

	sendMessage(conn, "search_complete", response)
	logger.Info("websocket search completed", "results", len(ranked), "elapsed_ms", elapsed)
}

func analyzeContentWithProgress(ctx context.Context, conn *websocket.Conn, prompt string, results []SearchResult, cfg AppConfig, logger *Logger) []SearchResult {
	type contentEval struct {
		idx         int
		content     string
		keep        bool
		fetchFailed bool
		err         error
	}

	resultsCh := make(chan contentEval, len(results))
	var eg errgroup.Group
	eg.SetLimit(3)

	completed := 0
	mu := sync.Mutex{}

	for i := range results {
		i := i
		eg.Go(func() error {
			contentCtx, contentCancel := context.WithTimeout(ctx, 20*time.Second)
			defer contentCancel()

			content, err := fetchPageContent(contentCtx, results[i].URL)
			if err != nil {
				logger.Error("content fetch failed", "error", err, "url", results[i].URL)
				resultsCh <- contentEval{idx: i, fetchFailed: true, err: err}
			} else {
				relevant, relErr := isContentRelevantToPrompt(contentCtx, prompt, results[i].Title, results[i].URL, content, cfg.OpenRouterAPIKey)
				if relErr != nil {
					logger.Error("content relevance evaluation failed", "error", relErr, "url", results[i].URL)
				}
				resultsCh <- contentEval{idx: i, content: content, keep: relErr == nil && relevant, err: relErr}
			}

			mu.Lock()
			completed++
			mu.Unlock()

			// Отправляем обновление прогресса
			sendStatus(conn, "analyzing_content", completed, len(results),
				"Проанализировано страниц: %d/%d", completed, len(results))

			return nil
		})
	}

	_ = eg.Wait()
	close(resultsCh)

	// Фильтруем результаты
	keepMap := make(map[int]bool, len(results))
	for r := range resultsCh {
		if r.fetchFailed {
			keepMap[r.idx] = false
		} else if r.err != nil {
			keepMap[r.idx] = true // graceful degrade
		} else {
			keepMap[r.idx] = r.keep
		}
	}

	filtered := make([]SearchResult, 0, len(results))
	for i, result := range results {
		if keep, ok := keepMap[i]; ok && keep {
			filtered = append(filtered, result)
		} else if !ok {
			filtered = append(filtered, result) // No evaluation result
		}
	}

	return filtered
}

func sendMessage(conn *websocket.Conn, msgType string, data interface{}) error {
	msg := WSMessage{
		Type: msgType,
		Data: data,
	}
	return conn.WriteJSON(msg)
}

func sendStatus(conn *websocket.Conn, stage string, progress, total int, format string, args ...interface{}) {
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	} else {
		message = format
	}

	status := WSSearchStatus{
		Stage:     stage,
		Progress:  progress,
		Total:     total,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
	sendMessage(conn, "status", status)
}

func sendError(conn *websocket.Conn, code, message, details string) {
	err := WSError{
		Code:    code,
		Message: message,
		Details: details,
	}
	sendMessage(conn, "error", err)
}
