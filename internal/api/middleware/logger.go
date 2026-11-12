package middleware

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey struct{}

type statusRecoder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader - Code(request) interception to generate a log
func (r *statusRecoder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Write for counting bytes
func (r *statusRecoder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Unwrap - Implementation http.ResponseController interface
func (r *statusRecoder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Logging
// Measures request duration
// Generates and reuses Request-ID
// Create a child logger with request_id
// Puts logger into context
// Decorates ResponseWriter
// Sets X-Request-ID in Response
// Calls next handler and write log with level
func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" || len(reqID) > 64 { // log size limit
			reqID = rand.Text()[:12]
		}
		l := log.With("request_id", reqID)
		ctx := context.WithValue(r.Context(), ctxKey{}, l)

		rec := &statusRecoder{ResponseWriter: w, status: http.StatusOK}
		rec.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(rec, r.WithContext(ctx))

		level := slog.LevelInfo
		switch {
		case rec.status >= 500:
			level = slog.LevelError
		case rec.status >= 400:
			level = slog.LevelWarn
		}
		l.LogAttrs(ctx, level, "http",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Int("bytes", rec.bytes),
			slog.Duration("took", time.Since(start)),
		)
	})
}

// L retrieves the logger from the context
func L(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
