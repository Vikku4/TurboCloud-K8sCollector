package httpserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	tclog "github.com/turbocloud-io/k8s-collector/internal/log"
)

// RequestLogger logs basic request info + status + duration.
// It also injects an X-Request-ID header.
func RequestLogger(logger *tclog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the ResponseWriter so we can see status / bytes.
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Simple request ID
			reqID := uuid.New().String()
			ww.Header().Set("X-Request-ID", reqID)

			// Call the next handler
			next.ServeHTTP(ww, r)

			// Log after response is written
			logger.Infof("%s %s %d %s req_id=%s",
				r.Method,
				r.URL.Path,
				ww.Status(),
				time.Since(start),
				reqID,
			)
		})
	}
}

// Recoverer uses chi's Recoverer to avoid panics crashing the server.
// You can later extend this to log stack traces with logger if you want.
func Recoverer(logger *tclog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return middleware.Recoverer(next)
	}
}
