package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type searxResultItem struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type searxResponse struct {
	Results []searxResultItem `json:"results"`
}

// logToFile записывает данные в файл логов
func logToFile(cfg AppConfig, data string) error {
	if !cfg.Debug.Enabled || !cfg.Debug.LogRequests {
		return nil
	}

	logFilePath := fmt.Sprintf("%s/searx_debug.log", cfg.Debug.LogFile)
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, data)
	_, err = file.WriteString(logEntry)
	return err
}

func searchSearx(ctx context.Context, cfg AppConfig, query string, engines []string) ([]SearchResult, error) {
	base := strings.TrimRight(cfg.Searx.URL, "/")
	endpoint := fmt.Sprintf("%s/search?q=%s&format=json&language=%s&locale=%s", base, url.QueryEscape(query), cfg.Searx.Language, cfg.Searx.Locale)
	if len(engines) > 0 {
		// SearxNG accepts engines as comma-separated list
		endpoint += "&engines=" + url.QueryEscape(strings.Join(engines, ","))
	}

	// Логируем запрос в JSON формате
	requestData := map[string]interface{}{
		"type":      "request",
		"method":    "GET",
		"url":       endpoint,
		"query":     query,
		"engines":   engines,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if requestJSON, err := json.MarshalIndent(requestData, "", "  "); err == nil {
		logToFile(cfg, string(requestJSON))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: cfg.Timeouts.SearxRequest}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа для логирования
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Логируем ответ в JSON формате
	var responseJSON interface{}
	json.Unmarshal(body, &responseJSON)

	responseData := map[string]interface{}{
		"type":        "response",
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        responseJSON,
		"timestamp":   time.Now().Format(time.RFC3339),
	}
	if respJSON, err := json.MarshalIndent(responseData, "", "  "); err == nil {
		logToFile(cfg, string(respJSON))
	}

	var sr searxResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, err
	}

	var out []SearchResult
	for i, r := range sr.Results {
		score := r.Score
		if score == 0 {
			// simple heuristic based on position
			score = 1.0 / float64(i+1)
		}
		out = append(out, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Score:   score,
		})
	}
	return out, nil
}
