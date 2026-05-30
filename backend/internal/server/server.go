package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"go.uber.org/fx"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/pkg/config"
)

// authRateLimitMiddleware limits requests to /api/v1/auth/* to 10 per minute
// per IP address to mitigate brute-force attacks.
func authRateLimitMiddleware() func(http.Handler) http.Handler {
	limiter := httprate.NewRateLimiter(10, time.Minute,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"title":"Too Many Requests","status":429,"detail":"rate limit exceeded, try again later"}`)
		}),
	)
	return func(next http.Handler) http.Handler {
		limited := limiter.Handler(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
				limited.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// jwtMiddleware protects all /api/v1/* routes except /api/v1/auth/*.
func jwtMiddleware(auth domain.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only protect API routes.
			if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}
			// Auth endpoints are always public.
			if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"title":"Unauthorized","status":401,"detail":"missing or invalid authorization header"}`)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if _, err := auth.ValidateToken(token); err != nil {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"title":"Unauthorized","status":401,"detail":"invalid or expired token"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewRouter creates the chi router with middleware and initialises the Huma API
// instance (which auto-generates the OpenAPI spec at /openapi.json and serves
// Swagger UI at /docs).
func NewRouter(cfg *config.Config, auth domain.AuthService) (*chi.Mux, huma.API) {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))
	router.Use(authRateLimitMiddleware())
	router.Use(jwtMiddleware(auth))

	// Health check (outside Huma so it doesn't appear in the OpenAPI spec)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	openAPIConfig := huma.DefaultConfig("RatioDash API", "1.0.0")
	openAPIConfig.DocsRenderer = huma.DocsRendererSwaggerUI
	if openAPIConfig.Components == nil {
		openAPIConfig.Components = &huma.Components{}
	}
	if openAPIConfig.Components.SecuritySchemes == nil {
		openAPIConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	openAPIConfig.Components.SecuritySchemes["bearerAuth"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT token returned by POST /api/v1/auth/login",
	}
	// All API operations require JWT by default; public auth routes override this.
	openAPIConfig.Security = []map[string][]string{{"bearerAuth": {}}}

	api := humachi.New(router, openAPIConfig)

	// Serve the embedded Vue SPA. In dev the dist/ placeholder has no
	// index.html, so static serving is skipped and Vite handles the frontend.
	if sub, err := fs.Sub(staticFiles, "dist"); err == nil {
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			fileServer := http.FileServer(http.FS(sub))
			router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
				if _, err := fs.Stat(sub, r.URL.Path[1:]); err == nil {
					fileServer.ServeHTTP(w, r)
				} else {
					http.ServeFileFS(w, r, sub, "index.html")
				}
			})
		}
	}

	return router, api
}

// Start registers the HTTP server lifecycle hooks with FX.
func Start(lc fx.Lifecycle, router *chi.Mux, cfg *config.Config) {
	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("Server listening on %s", cfg.ServerAddr)
			log.Printf("API docs:  http://%s/docs", cfg.ServerAddr)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

var Module = fx.Options(
	fx.Provide(NewRouter),
	fx.Invoke(Start),
)
