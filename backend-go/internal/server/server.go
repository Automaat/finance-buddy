// Package server wires the HTTP router and middleware for the Go backend.
//
// Kept in an internal package (rather than alongside main) so it can be
// exercised by tests without spinning up a full process and so future endpoint
// handlers have a stable home as they're cut over from Python.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Config holds the runtime knobs the server reads at startup.
//
// The caller (cmd/api) owns env-var loading and defaulting; New treats every
// field as already-resolved input. Empty values flow through unchanged so the
// behavior matches the Python backend during cutover.
type Config struct {
	// Addr is the listen address (host:port). Consumed by cmd/api when
	// constructing http.Server; not used by New itself.
	Addr string
	// CORSOrigins is the exact value of the CORS_ORIGINS env var. It's
	// split on "," verbatim (no trimming, no wildcard fallback) to match
	// the Python backend's `settings.cors_origins.split(",")`.
	CORSOrigins string
}

// New returns a chi router with the shared middleware stack and the only
// endpoint this skeleton ships: GET /health.
func New(cfg Config, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitOrigins(cfg.CORSOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	r.Get("/health", healthHandler(logger))

	return r
}

func healthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			logger.Error("encode health response", "err", err)
		}
	}
}

// splitOrigins mirrors the Python backend's `settings.cors_origins.split(",")`
// verbatim — no trimming, no wildcard fallback. Behavior parity matters
// during proxy-mediated cutover (the SvelteKit client sees exactly one of the
// two backends per request).
func splitOrigins(raw string) []string {
	return strings.Split(raw, ",")
}
