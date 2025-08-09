package main

import (
	"bytes"
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

func generateQueriesWithOpenRouter(prompt string, n int, apiKey string) ([]string, error) {
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
	httpReq, _ := http.NewRequest("POST", openRouterEndpoint, bytes.NewReader(b))
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
