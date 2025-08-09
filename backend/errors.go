package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AppError представляет структурированную ошибку приложения
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Предопределенные типы ошибок
var (
	ErrInvalidRequest = &AppError{
		Code:    "INVALID_REQUEST",
		Message: "Invalid request format",
		Status:  http.StatusBadRequest,
	}
	ErrMissingAPIKey = &AppError{
		Code:    "MISSING_API_KEY",
		Message: "OpenRouter API key is not configured",
		Status:  http.StatusInternalServerError,
	}
	ErrQueryGeneration = &AppError{
		Code:    "QUERY_GENERATION_FAILED",
		Message: "Failed to generate search queries",
		Status:  http.StatusInternalServerError,
	}
	ErrSearchFailed = &AppError{
		Code:    "SEARCH_FAILED",
		Message: "Search operation failed",
		Status:  http.StatusInternalServerError,
	}
	ErrContentFetch = &AppError{
		Code:    "CONTENT_FETCH_FAILED",
		Message: "Failed to fetch page content",
		Status:  http.StatusInternalServerError,
	}
)

// NewAppError создает новую ошибку приложения
func NewAppError(code, message, details string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Status:  status,
	}
}

// WrapError оборачивает обычную ошибку в AppError
func WrapError(base *AppError, err error) *AppError {
	return &AppError{
		Code:    base.Code,
		Message: base.Message,
		Details: err.Error(),
		Status:  base.Status,
	}
}

// ErrorResponse отправляет структурированный ответ об ошибке
func ErrorResponse(w http.ResponseWriter, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)

	response := struct {
		Error AppError `json:"error"`
	}{
		Error: *err,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// RecoveryMiddleware перехватывает панику и возвращает структурированную ошибку
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				appErr := &AppError{
					Code:    "INTERNAL_ERROR",
					Message: "Internal server error",
					Details: fmt.Sprintf("panic: %v", err),
					Status:  http.StatusInternalServerError,
				}
				ErrorResponse(w, appErr)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
