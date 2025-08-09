package main

import (
	"context"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func fetchPageContent(ctx context.Context, targetURL string) (string, error) {
	// Derive timeout from context if available
	timeout := 20 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		d := time.Until(deadline)
		if d > 0 {
			timeout = d
		}
	}

	article, err := readability.FromURL(targetURL, timeout)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(article.TextContent)
	if text == "" {
		text = strings.TrimSpace(article.Excerpt)
	}
	text = strings.Join(strings.Fields(text), " ")
	return text, nil
}
