package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

func fetchPageContent(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Limit to 64KB for memory safety
	limited := io.LimitReader(resp.Body, 64*1024)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(string(data))
	// Collapse whitespace
	text = strings.Join(strings.Fields(text), " ")
	return text, nil
}
