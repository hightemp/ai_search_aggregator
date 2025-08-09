package main

import (
	"fmt"
	"strings"
)

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// ValidateSearchRequest валидирует запрос поиска
func ValidateSearchRequest(req *SearchRequest) ValidationErrors {
	var errors ValidationErrors

	// Валидация prompt
	if strings.TrimSpace(req.Prompt) == "" {
		errors = append(errors, ValidationError{
			Field:   "prompt",
			Message: "prompt cannot be empty",
		})
	}

	if len(req.Prompt) > 1000 {
		errors = append(errors, ValidationError{
			Field:   "prompt",
			Message: "prompt cannot exceed 1000 characters",
		})
	}

	// Валидация settings
	if req.Settings.Queries < 1 || req.Settings.Queries > 20 {
		errors = append(errors, ValidationError{
			Field:   "settings.queries",
			Message: "queries must be between 1 and 20",
		})
	}

	// Валидация engines
	if len(req.Settings.Engines) > 0 {
		validEngines := map[string]bool{
			"google": true, "bing": true, "duckduckgo": true, "brave": true,
			"qwant": true, "yandex": true, "wikipedia": true, "github": true,
			"stackoverflow": true, "reddit": true, "youtube": true,
		}

		for _, engine := range req.Settings.Engines {
			if !validEngines[engine] {
				errors = append(errors, ValidationError{
					Field:   "settings.engines",
					Message: fmt.Sprintf("invalid engine: %s", engine),
				})
			}
		}

		if len(req.Settings.Engines) > 10 {
			errors = append(errors, ValidationError{
				Field:   "settings.engines",
				Message: "cannot specify more than 10 engines",
			})
		}
	}

	return errors
}

// SanitizeSearchRequest очищает и нормализует входные данные
func SanitizeSearchRequest(req *SearchRequest) {
	// Очистка prompt
	req.Prompt = strings.TrimSpace(req.Prompt)

	// Удаление дубликатов из engines
	if len(req.Settings.Engines) > 0 {
		engineSet := make(map[string]bool)
		var uniqueEngines []string
		for _, engine := range req.Settings.Engines {
			engine = strings.TrimSpace(strings.ToLower(engine))
			if engine != "" && !engineSet[engine] {
				engineSet[engine] = true
				uniqueEngines = append(uniqueEngines, engine)
			}
		}
		req.Settings.Engines = uniqueEngines
	}
}
