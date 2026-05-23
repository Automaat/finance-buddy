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

	"github.com/Automaat/finance-buddy/backend-go/internal/accounts"
	"github.com/Automaat/finance-buddy/backend-go/internal/aggregates"
	"github.com/Automaat/finance-buddy/backend-go/internal/assets"
	"github.com/Automaat/finance-buddy/backend-go/internal/auth"
	"github.com/Automaat/finance-buddy/backend-go/internal/bonds"
	bonusevents "github.com/Automaat/finance-buddy/backend-go/internal/bonus_events"
	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/config"
	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/dashboard"
	debtpayments "github.com/Automaat/finance-buddy/backend-go/internal/debt_payments"
	"github.com/Automaat/finance-buddy/backend-go/internal/debts"
	equitygrants "github.com/Automaat/finance-buddy/backend-go/internal/equity_grants"
	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/goals"
	"github.com/Automaat/finance-buddy/backend-go/internal/investment"
	"github.com/Automaat/finance-buddy/backend-go/internal/retirement"
	"github.com/Automaat/finance-buddy/backend-go/internal/salaries"
	"github.com/Automaat/finance-buddy/backend-go/internal/simulations"
	"github.com/Automaat/finance-buddy/backend-go/internal/snapshots"
	"github.com/Automaat/finance-buddy/backend-go/internal/transactions"
	"github.com/Automaat/finance-buddy/backend-go/internal/zus"
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
	// JWTSecret signs and verifies session tokens (FB_JWT_SECRET).
	JWTSecret string
	// CookieSecure marks the session cookie Secure (FB_COOKIE_SECURE).
	// Off by default so plain-HTTP local/LAN deploys work.
	CookieSecure bool
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
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitOrigins(cfg.CORSOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	r.Get("/health", healthHandler(logger))

	if deps.Pool != nil {
		registerRoutes(r, cfg, deps.Pool, logger)
	}

	return r
}

// registerRoutes wires the public auth endpoints, then gates every other
// /api route behind a valid session.
func registerRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	tokens := auth.NewTokenService(cfg.JWTSecret)
	authHandler := auth.NewHandler(auth.NewStore(pool), tokens, cfg.CookieSecure, logger)

	// Public — reachable without a session.
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)

	// Everything else requires authentication.
	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(tokens))
		r.Get("/api/auth/me", authHandler.Me)
		r.Get("/api/users", authHandler.ListOwners)
		r.With(auth.RequireAdmin).Get("/api/auth/users", authHandler.ListUsers)
		r.With(auth.RequireAdmin).Post("/api/auth/users", authHandler.CreateUser)
		r.With(auth.RequireAdmin).Put("/api/auth/users/{id}", authHandler.UpdateUser)
		registerAPIRoutes(r, pool, logger)
	})
}

func registerAPIRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	registerCoreRoutes(r, pool, logger)
	registerEquityRoutes(r, pool, logger)
	registerCPIAndPayrollRoutes(r, pool, logger)
	registerPortfolioRoutes(r, pool, logger)
	registerLedgerRoutes(r, pool, logger)
	registerDashboardRoutes(r, pool, logger)
}

func registerLedgerRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	txStore := transactions.NewStore(pool)
	txHandler := transactions.NewHandler(txStore, logger)
	r.Get("/api/accounts/{account_id}/transactions", txHandler.ListForAccount)
	r.Post("/api/accounts/{account_id}/transactions", txHandler.Create)
	r.Delete("/api/accounts/{account_id}/transactions/{transaction_id}", txHandler.Delete)
	r.Get("/api/transactions", txHandler.ListAll)
	r.Get("/api/transactions/counts", txHandler.Counts)
	r.Get("/api/transactions/types", txHandler.Types)

	dpStore := debtpayments.NewStore(pool)
	dpHandler := debtpayments.NewHandler(dpStore, logger)
	r.Get("/api/accounts/{account_id}/payments", dpHandler.ListForAccount)
	r.Post("/api/accounts/{account_id}/payments", dpHandler.Create)
	r.Delete("/api/accounts/{account_id}/payments/{payment_id}", dpHandler.Delete)
	r.Get("/api/payments", dpHandler.ListAll)
	r.Get("/api/payments/counts", dpHandler.Counts)

	debtsStore := debts.NewStore(pool)
	debtsHandler := debts.NewHandler(debtsStore, logger)
	r.Get("/api/debts", debtsHandler.List)
	r.Post("/api/debts", debtsHandler.CreateWithAccount)
	r.Post("/api/accounts/{account_id}/debts", debtsHandler.Create)
	r.Get("/api/debts/{id}", debtsHandler.Get)
	r.Put("/api/debts/{id}", debtsHandler.Update)
	r.Delete("/api/debts/{id}", debtsHandler.Delete)

	retHandler := retirement.NewHandler(retirement.NewStore(pool), logger)
	r.Get("/api/retirement/stats", retHandler.Stats)
	r.Get("/api/retirement/ppk-stats", retHandler.PPKStats)
	r.Post("/api/retirement/ppk-contributions/generate", retHandler.GeneratePPKContributions)
	r.Get("/api/retirement/limits/{year}", retHandler.LimitsForYear)
	r.Put("/api/retirement/limits/{year}/{wrapper}/{owner_user_id}", retHandler.UpsertLimit)

	invHandler := investment.NewHandler(investment.NewStore(pool), logger)
	r.Get("/api/investment/stock-stats", invHandler.StockStats)
	r.Get("/api/investment/bond-stats", invHandler.BondStats)

	simHandler := simulations.NewHandler(simulations.NewStore(pool), logger)
	r.Post("/api/simulations/mortgage-vs-invest", simHandler.MortgageVsInvest)
	r.Post("/api/simulations/retirement", simHandler.Retirement)
	r.Get("/api/simulations/prefill", simHandler.Prefill)
	r.Post("/api/simulations/monte-carlo", simHandler.MonteCarlo)
}

func registerDashboardRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	dashHandler := dashboard.NewHandler(dashboard.NewStore(pool), logger)
	r.Get("/api/dashboard", dashHandler.Get)
}

func registerCoreRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	cfgHandler := config.NewHandler(config.NewStore(pool), logger)
	r.Route("/api/config", func(r chi.Router) {
		r.Get("/", cfgHandler.Get)
		r.Put("/", cfgHandler.Put)
	})

	goalsHandler := goals.NewHandler(goals.NewStore(pool), logger)
	r.Route("/api/goals", func(r chi.Router) {
		r.Get("/", goalsHandler.List)
		r.Post("/", goalsHandler.Create)
		r.Get("/{id}", goalsHandler.Get)
		r.Put("/{id}", goalsHandler.Update)
		r.Delete("/{id}", goalsHandler.Delete)
	})
}

func registerEquityRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	valuationsStore := companyvaluations.NewStore(pool)
	valuationsHandler := companyvaluations.NewHandler(valuationsStore, logger)
	r.Route("/api/company-valuations", func(r chi.Router) {
		r.Get("/", valuationsHandler.List)
		r.Post("/", valuationsHandler.Create)
		r.Get("/{id}", valuationsHandler.Get)
		r.Patch("/{id}", valuationsHandler.Update)
		r.Delete("/{id}", valuationsHandler.Delete)
	})

	fxSvc := fx.NewService(pool, logger)
	bonusesHandler := bonusevents.NewHandler(bonusevents.NewStore(pool), fxSvc, logger)
	r.Route("/api/bonuses", func(r chi.Router) {
		r.Get("/", bonusesHandler.List)
		r.Post("/", bonusesHandler.Create)
		r.Get("/{id}", bonusesHandler.Get)
		r.Patch("/{id}", bonusesHandler.Update)
		r.Delete("/{id}", bonusesHandler.Delete)
	})

	grantsHandler := equitygrants.NewHandler(
		equitygrants.NewStore(pool), valuationsStore, fxSvc, logger,
	)
	r.Route("/api/equity-grants", func(r chi.Router) {
		r.Get("/", grantsHandler.List)
		r.Post("/", grantsHandler.Create)
		r.Get("/{id}", grantsHandler.Get)
		r.Patch("/{id}", grantsHandler.Update)
		r.Delete("/{id}", grantsHandler.Delete)
	})
}

func registerCPIAndPayrollRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	cpiStore := cpi.NewStore(pool)
	cpiHandler := cpi.NewHandler(cpiStore, cpi.NewGUSFetcher(), logger)
	r.Route("/api/cpi", func(r chi.Router) {
		r.Get("/series", cpiHandler.GetSeries)
		r.Post("/adjust", cpiHandler.Adjust)
		r.Post("/refresh", cpiHandler.Refresh)
	})

	salariesHandler := salaries.NewHandler(salaries.NewStore(pool), cpiStore, logger)
	r.Route("/api/salaries", func(r chi.Router) {
		r.Get("/", salariesHandler.List)
		r.Post("/", salariesHandler.Create)
		r.Get("/{id}", salariesHandler.Get)
		r.Patch("/{id}", salariesHandler.Update)
		r.Delete("/{id}", salariesHandler.Delete)
	})

	zusHandler := zus.NewHandler(zus.NewStore(pool), logger)
	r.Route("/api/zus", func(r chi.Router) {
		r.Post("/calculate", zusHandler.Calculate)
		r.Get("/prefill", zusHandler.Prefill)
	})
}

func registerPortfolioRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	aggregatesStore := aggregates.NewStore(pool)
	accountsHandler := accounts.NewHandler(accounts.NewStore(pool, aggregatesStore), logger)
	r.Route("/api/accounts", func(r chi.Router) {
		r.Get("/", accountsHandler.List)
		r.Post("/", accountsHandler.Create)
		r.Put("/{id}", accountsHandler.Update)
		r.Delete("/{id}", accountsHandler.Delete)
	})

	assetsHandler := assets.NewHandler(assets.NewStore(pool, aggregatesStore), logger)
	r.Route("/api/assets", func(r chi.Router) {
		r.Get("/", assetsHandler.List)
		r.Post("/", assetsHandler.Create)
		r.Put("/{id}", assetsHandler.Update)
		r.Delete("/{id}", assetsHandler.Delete)
	})

	snapshotsHandler := snapshots.NewHandler(snapshots.NewStore(pool, aggregatesStore), logger)
	r.Route("/api/snapshots", func(r chi.Router) {
		r.Get("/", snapshotsHandler.List)
		r.Post("/", snapshotsHandler.Create)
		r.Get("/{id}", snapshotsHandler.Get)
		r.Put("/{id}", snapshotsHandler.Update)
	})

	bondsHandler := bonds.NewHandler(bonds.NewStore(pool), cpi.NewStore(pool), logger)
	r.Route("/api/bonds", func(r chi.Router) {
		r.Get("/", bondsHandler.List)
		r.Post("/", bondsHandler.Create)
		r.Get("/{id}", bondsHandler.Get)
		r.Get("/{id}/ytm", bondsHandler.YTM)
		r.Put("/{id}", bondsHandler.Update)
		r.Delete("/{id}", bondsHandler.Delete)
	})
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
