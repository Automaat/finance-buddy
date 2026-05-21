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
	"github.com/jackc/pgx/v5/pgxpool"

	bonusevents "github.com/Automaat/finance-buddy/backend-go/internal/bonus_events"
	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/config"
	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	equitygrants "github.com/Automaat/finance-buddy/backend-go/internal/equity_grants"
	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/goals"
	"github.com/Automaat/finance-buddy/backend-go/internal/personas"
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

// Deps bundles the runtime dependencies the router needs to construct
// handlers (DB pool, etc). Optional fields default to no-op behavior so
// existing /health-only tests don't need a real pool.
type Deps struct {
	Pool *pgxpool.Pool
}

// New returns a chi router with the shared middleware stack and the endpoints
// cut over so far.
func New(cfg Config, logger *slog.Logger, deps Deps) http.Handler {
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

	if deps.Pool != nil {
		cfgHandler := config.NewHandler(config.NewStore(deps.Pool), logger)
		r.Route("/api/config", func(r chi.Router) {
			r.Get("/", cfgHandler.Get)
			r.Put("/", cfgHandler.Put)
		})

		personasHandler := personas.NewHandler(personas.NewStore(deps.Pool), logger)
		r.Route("/api/personas", func(r chi.Router) {
			r.Get("/", personasHandler.List)
			r.Post("/", personasHandler.Create)
			r.Put("/{id}", personasHandler.Update)
			r.Delete("/{id}", personasHandler.Delete)
		})

		goalsHandler := goals.NewHandler(goals.NewStore(deps.Pool), logger)
		r.Route("/api/goals", func(r chi.Router) {
			r.Get("/", goalsHandler.List)
			r.Post("/", goalsHandler.Create)
			r.Get("/{id}", goalsHandler.Get)
			r.Put("/{id}", goalsHandler.Update)
			r.Delete("/{id}", goalsHandler.Delete)
		})

		valuationsHandler := companyvaluations.NewHandler(
			companyvaluations.NewStore(deps.Pool), logger,
		)
		r.Route("/api/company-valuations", func(r chi.Router) {
			r.Get("/", valuationsHandler.List)
			r.Post("/", valuationsHandler.Create)
			r.Get("/{id}", valuationsHandler.Get)
			r.Patch("/{id}", valuationsHandler.Update)
			r.Delete("/{id}", valuationsHandler.Delete)
		})

		fxSvc := fx.NewService(deps.Pool, logger)
		bonusesHandler := bonusevents.NewHandler(
			bonusevents.NewStore(deps.Pool), fxSvc, logger,
		)
		r.Route("/api/bonuses", func(r chi.Router) {
			r.Get("/", bonusesHandler.List)
			r.Post("/", bonusesHandler.Create)
			r.Get("/{id}", bonusesHandler.Get)
			r.Patch("/{id}", bonusesHandler.Update)
			r.Delete("/{id}", bonusesHandler.Delete)
		})

		grantsHandler := equitygrants.NewHandler(
			equitygrants.NewStore(deps.Pool),
			companyvaluations.NewStore(deps.Pool),
			fxSvc,
			logger,
		)
		r.Route("/api/equity-grants", func(r chi.Router) {
			r.Get("/", grantsHandler.List)
			r.Post("/", grantsHandler.Create)
			r.Get("/{id}", grantsHandler.Get)
			r.Patch("/{id}", grantsHandler.Update)
			r.Delete("/{id}", grantsHandler.Delete)
		})

		cpiHandler := cpi.NewHandler(cpi.NewStore(deps.Pool), cpi.NewGUSFetcher(), logger)
		r.Route("/api/cpi", func(r chi.Router) {
			r.Get("/series", cpiHandler.GetSeries)
			r.Post("/adjust", cpiHandler.Adjust)
			r.Post("/refresh", cpiHandler.Refresh)
		})
	}

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
