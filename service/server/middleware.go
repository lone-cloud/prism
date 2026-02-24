package server

import (
	"log/slog"
	"net/http"
	"time"

	"prism/service/util"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

var noisyPaths = map[string]bool{
	"/.well-known/appspecific/com.chrome.devtools.json": true,
	"/health": true,
}

func authMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !util.VerifyAPIKey(r, apiKey) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Prism Admin - Username: any, Password: API_KEY"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			if !noisyPaths[r.URL.Path] {
				logger.Debug("HTTP request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration", time.Since(start),
					"ip", util.GetClientIP(r),
				)
			}
		})
	}
}

func securityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://api.qrserver.com; form-action 'self'; frame-ancestors 'none'; object-src 'none'")

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitMiddleware(rps int) func(http.Handler) http.Handler {
	rl := httprate.LimitByIP(rps, time.Second)
	return func(next http.Handler) http.Handler {
		limited := rl(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if util.IsLocalhost(util.GetClientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}
			limited.ServeHTTP(w, r)
		})
	}
}

func maxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
