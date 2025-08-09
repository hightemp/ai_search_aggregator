package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func searchSearx(ctx context.Context, searxBase, query string, engines []string) ([]SearchResult, error) {
	base := strings.TrimRight(searxBase, "/")
	endpoint := fmt.Sprintf("%s/search?q=%s&format=json", base, url.QueryEscape(query))
	if len(engines) > 0 {
		// SearxNG accepts engines as comma-separated list
		endpoint += "&engines=" + url.QueryEscape(strings.Join(engines, ","))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sr searxResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
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
