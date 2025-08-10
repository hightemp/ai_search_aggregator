package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// AppConfig holds runtime configuration derived from environment variables.
type AppConfig struct {
	Server            ServerConfig
	OpenRouter        OpenRouterConfig
	Searx             SearxConfig
	WebSocket         WebSocketConfig
	Search            SearchConfig
	Content           ContentConfig
	Validation        ValidationConfig
	Timeouts          TimeoutConfig
	Limits            LimitsConfig
	Debug             DebugConfig
}

type ServerConfig struct {
	Port string
}

type OpenRouterConfig struct {
	APIKey            string
	Model             string
	QueryGenMaxTokens int
	FilterMaxTokens   int
	ContentMaxTokens  int
	Endpoint          string
}

type SearxConfig struct {
	URL      string
	Language string
	Locale   string
}

type WebSocketConfig struct {
	MaxConnections     int
	MessageTimeout     time.Duration
	SearchTimeout      time.Duration
	MaxMessageSize     int64
	AllowedOrigins     []string
	EnableOriginCheck  bool
}

type SearchConfig struct {
	DefaultQueryCount     int
	ContentModeDefault    bool
	MaxConcurrentQueries  int
	MaxConcurrentContent  int
	MaxConcurrentFilter   int
	MaxResultsToEvaluate  int
	MaxResultsToProcess   int
}

type ContentConfig struct {
	MaxContentLength int
	TruncationLength int
}

type ValidationConfig struct {
	MaxPromptLength   int
	MaxQueryCount     int
	MaxEngineCount    int
	SupportedEngines  []string
}

type TimeoutConfig struct {
	HTTPClient       time.Duration
	SearxRequest     time.Duration
	ContentFetch     time.Duration
	OpenRouterAPI    time.Duration
	QueryGeneration  time.Duration
	AIRelevance      time.Duration
	ContentRelevance time.Duration
}

type LimitsConfig struct {
	MaxSearchResults  int
	MaxItemsToFilter  int
	MaxContentItems   int
}

type DebugConfig struct {
	Enabled     bool
	LogRequests bool
	LogFile     string
}

func loadConfig() AppConfig {
	cfg := AppConfig{
		Server: ServerConfig{
			Port: getenv("PORT", "8080"),
		},
		OpenRouter: OpenRouterConfig{
			APIKey:            os.Getenv("OPENROUTER_API_KEY"),
			Model:             getenv("OPENROUTER_MODEL", "openai/gpt-4o-mini"),
			QueryGenMaxTokens: atoi(getenv("OPENROUTER_QUERY_MAX_TOKENS", "256"), 256),
			FilterMaxTokens:   atoi(getenv("OPENROUTER_FILTER_MAX_TOKENS", "64"), 64),
			ContentMaxTokens:  atoi(getenv("OPENROUTER_CONTENT_MAX_TOKENS", "4"), 4),
			Endpoint:          getenv("OPENROUTER_ENDPOINT", "https://openrouter.ai/api/v1/chat/completions"),
		},
		Searx: SearxConfig{
			URL:      getenv("SEARX_URL", "http://searx:8080"),
			Language: getenv("SEARX_LANGUAGE", "en"),
			Locale:   getenv("SEARX_LOCALE", "en-US"),
		},
		WebSocket: WebSocketConfig{
			MaxConnections:    atoi(getenv("WEBSOCKET_MAX_CONNECTIONS", "100"), 100),
			MessageTimeout:    parseDuration(getenv("WEBSOCKET_MESSAGE_TIMEOUT", "30s"), 30*time.Second),
			SearchTimeout:     parseDuration(getenv("WEBSOCKET_SEARCH_TIMEOUT", "10m"), 10*time.Minute),
			MaxMessageSize:    int64(atoi(getenv("WEBSOCKET_MAX_MESSAGE_SIZE", "65536"), 65536)), // 64KB
			AllowedOrigins:    parseStringSlice(getenv("WEBSOCKET_ALLOWED_ORIGINS", "")),
			EnableOriginCheck: getenv("WEBSOCKET_ENABLE_ORIGIN_CHECK", "false") == "true",
		},
		Search: SearchConfig{
			DefaultQueryCount:     atoi(getenv("SEARCH_DEFAULT_QUERY_COUNT", "5"), 5),
			ContentModeDefault:    getenv("SEARCH_CONTENT_MODE_DEFAULT", "false") == "true",
			MaxConcurrentQueries:  atoi(getenv("SEARCH_MAX_CONCURRENT_QUERIES", "5"), 5),
			MaxConcurrentContent:  atoi(getenv("SEARCH_MAX_CONCURRENT_CONTENT", "3"), 3),
			MaxConcurrentFilter:   atoi(getenv("SEARCH_MAX_CONCURRENT_FILTER", "3"), 3),
			MaxResultsToEvaluate:  atoi(getenv("SEARCH_MAX_RESULTS_TO_EVALUATE", "50"), 50),
			MaxResultsToProcess:   atoi(getenv("SEARCH_MAX_RESULTS_TO_PROCESS", "100"), 100),
		},
		Content: ContentConfig{
			MaxContentLength: atoi(getenv("CONTENT_MAX_LENGTH", "10000"), 10000),
			TruncationLength: atoi(getenv("CONTENT_TRUNCATION_LENGTH", "3500"), 3500),
		},
		Validation: ValidationConfig{
			MaxPromptLength: atoi(getenv("VALIDATION_MAX_PROMPT_LENGTH", "1000"), 1000),
			MaxQueryCount:   atoi(getenv("VALIDATION_MAX_QUERY_COUNT", "20"), 20),
			MaxEngineCount:  atoi(getenv("VALIDATION_MAX_ENGINE_COUNT", "10"), 10),
			SupportedEngines: parseStringSlice(getenv("VALIDATION_SUPPORTED_ENGINES", 
				"google,bing,duckduckgo,brave,qwant,yandex,wikipedia,github,stackoverflow,reddit,youtube")),
		},
		Timeouts: TimeoutConfig{
			HTTPClient:       parseDuration(getenv("TIMEOUT_HTTP_CLIENT", "30s"), 30*time.Second),
			SearxRequest:     parseDuration(getenv("TIMEOUT_SEARX_REQUEST", "20s"), 20*time.Second),
			ContentFetch:     parseDuration(getenv("TIMEOUT_CONTENT_FETCH", "20s"), 20*time.Second),
			OpenRouterAPI:    parseDuration(getenv("TIMEOUT_OPENROUTER_API", "60s"), 60*time.Second),
			QueryGeneration:  parseDuration(getenv("TIMEOUT_QUERY_GENERATION", "60s"), 60*time.Second),
			AIRelevance:      parseDuration(getenv("TIMEOUT_AI_RELEVANCE", "30s"), 30*time.Second),
			ContentRelevance: parseDuration(getenv("TIMEOUT_CONTENT_RELEVANCE", "30s"), 30*time.Second),
		},
		Limits: LimitsConfig{
			MaxSearchResults: atoi(getenv("LIMITS_MAX_SEARCH_RESULTS", "100"), 100),
			MaxItemsToFilter: atoi(getenv("LIMITS_MAX_ITEMS_TO_FILTER", "30"), 30),
			MaxContentItems:  atoi(getenv("LIMITS_MAX_CONTENT_ITEMS", "20"), 20),
		},
		Debug: DebugConfig{
			Enabled:     getenv("DEBUG", "false") == "true",
			LogRequests: getenv("DEBUG_LOG_REQUESTS", "true") == "true",
			LogFile:     getenv("DEBUG_LOG_FILE", "/tmp"),
		},
	}
	
	// Validate configuration
	if cfg.WebSocket.EnableOriginCheck && len(cfg.WebSocket.AllowedOrigins) == 0 {
		cfg.WebSocket.AllowedOrigins = []string{"localhost", "127.0.0.1"}
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

func parseDuration(s string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return def
}

func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	items := strings.Split(s, ",")
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
