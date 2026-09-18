package http

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const WebhookSecretHeader = "X-Gform-Secret"

func muxVar(r *http.Request, name string) string { return mux.Vars(r)[name] }

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
		)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("паника в обработчике", "recover", rec, "path", r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "внутренняя ошибка"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func webhookAuth(secret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			given := strings.TrimSpace(r.Header.Get(WebhookSecretHeader))
			if subtle.ConstantTimeCompare([]byte(given), []byte(secret)) != 1 {
				writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "неверный секрет"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
