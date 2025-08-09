package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeSearxResp struct {
	Results []searxResultItem `json:"results"`
}

func TestSearchSearx(t *testing.T) {
	resp := fakeSearxResp{
		Results: []searxResultItem{
			{Title: "Title A", URL: "https://a.com", Content: "Snippet A", Score: 0.9},
			{Title: "Title B", URL: "https://b.com", Content: "Snippet B", Score: 0.8},
		},
	}
	data, _ := json.Marshal(resp)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	results, err := searchSearx(ts.URL, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://a.com" {
		t.Fatalf("unexpected first URL: %s", results[0].URL)
	}
}
