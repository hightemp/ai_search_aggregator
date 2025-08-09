package main

import (
	"os"
	"strconv"
)

// AppConfig holds runtime configuration derived from environment variables.
type AppConfig struct {
	Port               string
	OpenRouterAPIKey   string
	SearxURL           string
	DefaultQueryCount  int
	ContentModeDefault bool
}

func loadConfig() AppConfig {
	cfg := AppConfig{
		Port:               getenv("PORT", "8080"),
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		SearxURL:           getenv("SEARX_URL", "http://searx:8080"),
		DefaultQueryCount:  atoi(getenv("DEFAULT_QUERY_COUNT", "5"), 5),
		ContentModeDefault: getenv("CONTENT_MODE_DEFAULT", "false") == "true",
	}
	return cfg
}

func getenv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func atoi(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
