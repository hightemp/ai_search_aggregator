package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

type Logger struct {
	*slog.Logger
}

type RequestLogger struct {
	logger *Logger
	rid    string
}

// NewLogger создает новый структурированный логгер
func NewLogger() *Logger {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithRequestID создает логгер с привязанным request ID
func (l *Logger) WithRequestID(rid string) *RequestLogger {
	return &RequestLogger{
		logger: l,
		rid:    rid,
	}
}

// Info логирует информационное сообщение с request ID
func (rl *RequestLogger) Info(msg string, args ...any) {
	args = append([]any{"rid", rl.rid}, args...)
	rl.logger.Info(msg, args...)
}

// Error логирует ошибку с request ID
func (rl *RequestLogger) Error(msg string, err error, args ...any) {
	args = append([]any{"rid", rl.rid, "error", err}, args...)
	rl.logger.Error(msg, args...)
}

// Debug логирует отладочное сообщение с request ID
func (rl *RequestLogger) Debug(msg string, args ...any) {
	args = append([]any{"rid", rl.rid}, args...)
	rl.logger.Debug(msg, args...)
}

// WithDuration добавляет информацию о времени выполнения
func (rl *RequestLogger) WithDuration(start time.Time) *RequestLogger {
	return &RequestLogger{
		logger: &Logger{
			Logger: rl.logger.Logger.With("duration_ms", time.Since(start).Milliseconds()),
		},
		rid: rl.rid,
	}
}

// LoggingMiddleware добавляет логирование HTTP запросов
func LoggingMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rid := generateRequestID()

			// Добавляем request ID в контекст
			type requestIDKey string
			const ridKey requestIDKey = "request_id"
			ctx := context.WithValue(r.Context(), ridKey, rid)
			r = r.WithContext(ctx)

			reqLogger := logger.WithRequestID(rid)
			reqLogger.Info("request_start",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
			)

			// Создаем wrapper для response writer чтобы захватить status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(wrapped, r)

			reqLogger.WithDuration(start).Info("request_end",
				"status", wrapped.statusCode,
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack реализует интерфейс http.Hijacker для поддержки WebSocket
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

// generateRequestID создает уникальный ID для запроса
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
