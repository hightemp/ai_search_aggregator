package main

import (
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

func searchSearx(searxBase, query string) ([]SearchResult, error) {
	endpoint := fmt.Sprintf("%s/search?q=%s&format=json", strings.TrimRight(searxBase, "/"), url.QueryEscape(query))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(endpoint)
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
