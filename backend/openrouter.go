package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type openRouterRequest struct {
	Model     string        `json:"model"`
	Messages  []openMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type openMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message openMessage `json:"message"`
	} `json:"choices"`
}

var openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"

// Debug logging function for OpenRouter requests and responses
func logOpenRouterRequest(apiType string, request openRouterRequest, response *openRouterResponse, err error, statusCode int) {

	if os.Getenv("DEBUG") != "true" {
		return
	}

	// Log to both console and file
	consoleLogger := NewLogger()

	// Create or open log file in /tmp
	logFileName := fmt.Sprintf("/tmp/openrouter_debug_%s.log", time.Now().Format("2006-01-02"))
	logFile, fileErr := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if fileErr != nil {
		consoleLogger.Error("failed to open openrouter log file", "error", fileErr, "file", logFileName)
		return
	}
	defer logFile.Close()

	// Create file logger
	fileLogger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	timestamp := time.Now().Format(time.RFC3339)

	// Log request to both console and file
	requestJSON, _ := json.MarshalIndent(request, "", "  ")

	// Console log (short version)
	consoleLogger.Info("openrouter_request_debug",
		"api_type", apiType,
		"endpoint", openRouterEndpoint,
		"model", request.Model,
	)

	// File log (detailed version)
	fileLogger.Info("openrouter_request_detailed",
		"timestamp", timestamp,
		"api_type", apiType,
		"endpoint", openRouterEndpoint,
		"model", request.Model,
		"request_body", string(requestJSON),
	)

	// Log response
	if err != nil {
		// Console log
		consoleLogger.Error("openrouter_response_error",
			"api_type", apiType,
			"error", err.Error(),
			"status_code", statusCode,
		)

		// File log
		fileLogger.Error("openrouter_response_error_detailed",
			"timestamp", timestamp,
			"api_type", apiType,
			"error", err.Error(),
			"status_code", statusCode,
		)
	} else if response != nil {
		responseJSON, _ := json.MarshalIndent(response, "", "  ")

		// Console log (short version)
		consoleLogger.Info("openrouter_response_debug",
			"api_type", apiType,
			"status_code", statusCode,
			"choices_count", len(response.Choices),
		)

		// File log (detailed version)
		fileLogger.Info("openrouter_response_detailed",
			"timestamp", timestamp,
			"api_type", apiType,
			"status_code", statusCode,
			"choices_count", len(response.Choices),
			"response_body", string(responseJSON),
		)

		// Log the actual content separately for easier reading
		if len(response.Choices) > 0 {
			content := response.Choices[0].Message.Content

			// Console log
			consoleLogger.Info("openrouter_content_debug",
				"api_type", apiType,
				"content", content,
			)

			// File log
			fileLogger.Info("openrouter_content_detailed",
				"timestamp", timestamp,
				"api_type", apiType,
				"content", content,
			)
		}
	}
}

func generateQueriesWithOpenRouter(ctx context.Context, prompt string, n int, apiKey string) ([]string, error) {
	if apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY not set")
	}

	systemPrompt := fmt.Sprintf("You are a search assistant. Generate %d distinct web-search queries that would best answer the user's question. Respond with each query on a new line and nothing else.", n)

	reqBody := openRouterRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []openMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 256,
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", openRouterEndpoint, bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Title", "AI Search Aggregator")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)

	// Log request always
	logOpenRouterRequest("query_generation", reqBody, nil, err, 0)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Log error response
		logOpenRouterRequest("query_generation", reqBody, nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return nil, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		logOpenRouterRequest("query_generation", reqBody, nil, err, resp.StatusCode)
		return nil, err
	}

	// Log successful response
	logOpenRouterRequest("query_generation", reqBody, &orResp, nil, resp.StatusCode)

	if len(orResp.Choices) == 0 {
		return nil, errors.New("no choices returned from openrouter")
	}

	content := orResp.Choices[0].Message.Content
	lines := strings.Split(content, "\n")
	var queries []string
	for _, l := range lines {
		l = strings.TrimSpace(l)

		// Удаляем нумерацию в начале строки (1., 2., 1), 2), и т.д.)
		l = strings.TrimSpace(strings.TrimPrefix(l, "-"))

		// Удаляем различные варианты нумерации
		for i := 1; i <= 20; i++ {
			prefixes := []string{
				fmt.Sprintf("%d. ", i),
				fmt.Sprintf("%d) ", i),
				fmt.Sprintf("%d.", i),
				fmt.Sprintf("%d)", i),
			}
			for _, prefix := range prefixes {
				if strings.HasPrefix(l, prefix) {
					l = strings.TrimSpace(l[len(prefix):])
					break
				}
			}
		}

		// Удаляем кавычки в начале и конце
		l = strings.Trim(l, `"'`)

		if l != "" {
			queries = append(queries, l)
		}
	}
	if len(queries) > n {
		queries = queries[:n]
	}
	return queries, nil
}

// filterByAIRelevance uses the LLM to classify which search results are relevant to the user's prompt.
// It returns only those predicted as relevant while preserving order.
func filterByAIRelevance(ctx context.Context, prompt string, results []SearchResult, apiKey string) ([]SearchResult, error) {
	if apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY not set")
	}
	if len(results) == 0 {
		return results, nil
	}

	// Limit evaluated items to bound token usage
	const maxItems = 30
	end := len(results)
	if end > maxItems {
		end = maxItems
	}
	subset := results[:end]

	systemPrompt := "You are a strict relevance judge. For each item decide if it is relevant to the user's query based on title and snippet. Output ONLY a JSON array of 0 or 1 for each item in order."

	var sb strings.Builder
	sb.WriteString("User query:\n")
	sb.WriteString(prompt)
	sb.WriteString("\n\nItems to judge (keep order):\n")
	for i, it := range subset {
		sb.WriteString(fmt.Sprintf("%d) Title: %s\nURL: %s\nSnippet: %s\n\n", i+1, strings.TrimSpace(it.Title), strings.TrimSpace(it.URL), truncateForLLM(it.Snippet, 700)))
	}
	userPrompt := sb.String()

	reqBody := openRouterRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []openMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 64,
	}

	payload, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", openRouterEndpoint, bytes.NewReader(payload))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Title", "AI Relevance Filter")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)

	// Log request always
	logOpenRouterRequest("ai_relevance_filter", reqBody, nil, err, 0)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Log error response
		logOpenRouterRequest("ai_relevance_filter", reqBody, nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return nil, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		logOpenRouterRequest("ai_relevance_filter", reqBody, nil, err, resp.StatusCode)
		return nil, err
	}

	// Log successful response
	logOpenRouterRequest("ai_relevance_filter", reqBody, &orResp, nil, resp.StatusCode)

	if len(orResp.Choices) == 0 {
		return nil, errors.New("no choices returned from openrouter")
	}

	content := strings.TrimSpace(orResp.Choices[0].Message.Content)
	start := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if start == -1 || endIdx == -1 || endIdx <= start {
		return nil, fmt.Errorf("unexpected classifier response: %q", content)
	}
	jsonPart := content[start : endIdx+1]

	var ints []int
	if err := json.Unmarshal([]byte(jsonPart), &ints); err != nil {
		var bools []bool
		if err2 := json.Unmarshal([]byte(jsonPart), &bools); err2 != nil {
			return nil, fmt.Errorf("failed to parse classifier JSON: %v", err)
		}
		for _, v := range bools {
			if v {
				ints = append(ints, 1)
			} else {
				ints = append(ints, 0)
			}
		}
	}

	if len(ints) < len(subset) {
		pad := make([]int, len(subset)-len(ints))
		ints = append(ints, pad...)
	}

	filtered := make([]SearchResult, 0, len(results))
	for i, it := range subset {
		if i < len(ints) && ints[i] == 1 {
			filtered = append(filtered, it)
		}
	}
	if len(results) > len(subset) {
		filtered = append(filtered, results[len(subset):]...)
	}
	return filtered, nil
}

func truncateForLLM(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// isContentRelevantToPrompt judges a single page content for relevance to the user's query.
// Returns true if relevant, false otherwise.
func isContentRelevantToPrompt(ctx context.Context, prompt, title, url, content, apiKey string) (bool, error) {
	if apiKey == "" {
		return false, errors.New("OPENROUTER_API_KEY not set")
	}
	systemPrompt := "You are a strict binary relevance judge. Answer with a single character: 1 if the page content is relevant to the user's query, 0 if not. No explanation."

	userPrompt := strings.Builder{}
	userPrompt.WriteString("User query:\n")
	userPrompt.WriteString(prompt)
	userPrompt.WriteString("\n\nPage title:\n")
	userPrompt.WriteString(strings.TrimSpace(title))
	userPrompt.WriteString("\nURL:\n")
	userPrompt.WriteString(strings.TrimSpace(url))
	userPrompt.WriteString("\n\nPage content (may be truncated):\n")
	userPrompt.WriteString(truncateForLLM(content, 3500))

	reqBody := openRouterRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []openMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt.String()},
		},
		MaxTokens: 4,
	}

	payload, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", openRouterEndpoint, bytes.NewReader(payload))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Title", "AI Single Content Relevance")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)

	// Log request always
	logOpenRouterRequest("content_relevance", reqBody, nil, err, 0)

	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Log error response
		logOpenRouterRequest("content_relevance", reqBody, nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return false, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		logOpenRouterRequest("content_relevance", reqBody, nil, err, resp.StatusCode)
		return false, err
	}

	// Log successful response
	logOpenRouterRequest("content_relevance", reqBody, &orResp, nil, resp.StatusCode)

	if len(orResp.Choices) == 0 {
		return false, errors.New("no choices returned from openrouter")
	}
	ans := strings.TrimSpace(orResp.Choices[0].Message.Content)
	// Normalize
	ansLower := strings.ToLower(ans)
	if strings.HasPrefix(ans, "1") || ans == "1" || ansLower == "true" || strings.HasPrefix(ansLower, "yes") {
		return true, nil
	}
	return false, nil
}
