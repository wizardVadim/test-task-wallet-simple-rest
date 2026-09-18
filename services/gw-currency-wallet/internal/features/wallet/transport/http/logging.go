package wallet_http

import (
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *responseRecorder) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	if code >= 200 || code == http.StatusSwitchingProtocols {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *responseRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Logging records request outcomes without bodies, query parameters or headers.
func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		completed := false
		defer func() {
			status := recorder.status
			level := slog.LevelInfo
			if !completed {
				status = http.StatusInternalServerError
			}
			if status == 0 {
				status = http.StatusOK
			}
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 || r.Context().Err() != nil {
				level = slog.LevelWarn
			}
			logger.Log(r.Context(), level, "http request completed", "method", r.Method, "route", r.Pattern, "status", status, "duration_ms", float64(time.Since(start).Microseconds())/1000, "canceled", r.Context().Err() != nil)
		}()
		next.ServeHTTP(recorder, r)
		completed = true
	})
}
