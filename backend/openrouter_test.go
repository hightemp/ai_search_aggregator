package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeORResp struct {
	Choices []struct {
		Message openMessage `json:"message"`
	} `json:"choices"`
}

func TestGenerateQueriesWithOpenRouter(t *testing.T) {
	// Prepare fake response content
	content := "query one\nquery two\nquery three"
	respObj := fakeORResp{
		Choices: []struct {
			Message openMessage `json:"message"`
		}{{Message: openMessage{Role: "assistant", Content: content}}},
	}
	data, _ := json.Marshal(respObj)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	// Override endpoint
	old := openRouterEndpoint
	openRouterEndpoint = ts.URL
	defer func() { openRouterEndpoint = old }()

	queries, err := generateQueriesWithOpenRouter(context.Background(), "some prompt", 3, "fake-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(queries))
	}
	if queries[0] != "query one" {
		t.Fatalf("unexpected first query: %s", queries[0])
	}
}
