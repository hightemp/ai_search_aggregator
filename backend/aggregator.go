package main

import (
	"sort"
)

// deduplicateAndRank merges duplicate URLs and sorts by score descending.
func deduplicateAndRank(in []SearchResult) []SearchResult {
	m := make(map[string]SearchResult)
	for _, r := range in {
		if existing, ok := m[r.URL]; ok {
			// keep the higher score & longer snippet
			if r.Score > existing.Score {
				m[r.URL] = r
			}
		} else {
			m[r.URL] = r
		}
	}
	var out []SearchResult
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}
