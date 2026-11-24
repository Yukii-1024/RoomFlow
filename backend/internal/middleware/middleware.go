// Package middleware provides logging, panic recovery, CORS and JWT auth.
package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/auth"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// Auth validates the Bearer token and stores the principal in the request context.
func Auth(tokens *auth.Manager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				httpx.WriteError(w, httpx.Unauthorized("请先登录"))
				return
			}
			claims, err := tokens.Parse(strings.TrimSpace(token))
			if err != nil {
				httpx.WriteError(w, httpx.Unauthorized("登录状态已失效，请重新登录"))
				return
			}
			p := httpx.Principal{UserID: claims.UserID(), Role: claims.Role}
			next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), p)))
		})
	}
}

func RequireRole(role string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := httpx.PrincipalFrom(r.Context())
			if !ok {
				httpx.WriteError(w, httpx.Unauthorized("请先登录"))
				return
			}
			if p.Role != role {
				httpx.WriteError(w, httpx.Forbidden("当前账号无权执行该操作"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(allowedOrigin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowedOrigin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start).String(),
			)
		})
	}
}

func Recoverer(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "panic", recovered, "path", r.URL.Path)
					httpx.WriteError(w, httpx.Internal(nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
