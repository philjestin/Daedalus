package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
)

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestLogger is middleware that logs HTTP requests with timing and correlation IDs.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate or extract request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()[:8] // Short ID for readability
		}

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// Wrap response writer to capture status
		wrapped := wrapResponseWriter(w)

		// Log request start (debug level to reduce noise)
		slog.Debug("request started",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		level := slog.LevelInfo
		if wrapped.status >= 500 {
			level = slog.LevelError
		} else if wrapped.status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, "request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// LogError logs an error with request context.
func LogError(ctx context.Context, msg string, err error, args ...any) {
	allArgs := make([]any, 0, len(args)+4)
	allArgs = append(allArgs, "error", err)
	if requestID := GetRequestID(ctx); requestID != "" {
		allArgs = append(allArgs, "request_id", requestID)
	}
	allArgs = append(allArgs, args...)
	slog.Error(msg, allArgs...)
}

// LogInfo logs an info message with request context.
func LogInfo(ctx context.Context, msg string, args ...any) {
	allArgs := make([]any, 0, len(args)+2)
	if requestID := GetRequestID(ctx); requestID != "" {
		allArgs = append(allArgs, "request_id", requestID)
	}
	allArgs = append(allArgs, args...)
	slog.Info(msg, allArgs...)
}

// LogWarn logs a warning with request context.
func LogWarn(ctx context.Context, msg string, args ...any) {
	allArgs := make([]any, 0, len(args)+2)
	if requestID := GetRequestID(ctx); requestID != "" {
		allArgs = append(allArgs, "request_id", requestID)
	}
	allArgs = append(allArgs, args...)
	slog.Warn(msg, allArgs...)
}
