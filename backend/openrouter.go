package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return nil, err
	}

	if len(orResp.Choices) == 0 {
		return nil, errors.New("no choices returned from openrouter")
	}

	content := orResp.Choices[0].Message.Content
	lines := strings.Split(content, "\n")
	var queries []string
	for _, l := range lines {
		l = strings.TrimSpace(strings.TrimPrefix(l, "-"))
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
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return nil, err
	}
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
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("openrouter status %d: %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return false, err
	}
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
