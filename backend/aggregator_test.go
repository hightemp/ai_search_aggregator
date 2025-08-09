package main

import "testing"

func TestDeduplicateAndRank(t *testing.T) {
	in := []SearchResult{
		{URL: "https://a.com", Score: 0.5},
		{URL: "https://b.com", Score: 0.9},
		{URL: "https://a.com", Score: 0.8}, // duplicate with higher score
	}
	out := deduplicateAndRank(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 unique results, got %d", len(out))
	}
	if out[0].URL != "https://b.com" {
		t.Fatalf("expected first result to be b.com with highest score")
	}
	if out[1].URL != "https://a.com" || out[1].Score != 0.8 {
		t.Fatalf("expected deduplicated a.com with score 0.8")
	}
}
