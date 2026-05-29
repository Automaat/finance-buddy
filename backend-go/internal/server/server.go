// Package server wires the HTTP router and middleware for the Go backend.
//
// Kept in an internal package (rather than alongside main) so it can be
// exercised by tests without spinning up a full process and so future endpoint
// handlers have a stable home as they're cut over from Python.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/finance-buddy/backend-go/internal/accounts"
	"github.com/Automaat/finance-buddy/backend-go/internal/aggregates"
	"github.com/Automaat/finance-buddy/backend-go/internal/allocation"
	"github.com/Automaat/finance-buddy/backend-go/internal/assets"
	"github.com/Automaat/finance-buddy/backend-go/internal/auth"
	"github.com/Automaat/finance-buddy/backend-go/internal/bondrates"
	"github.com/Automaat/finance-buddy/backend-go/internal/bonds"
	bonusevents "github.com/Automaat/finance-buddy/backend-go/internal/bonus_events"
	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/config"
	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/dashboard"
	debtpayments "github.com/Automaat/finance-buddy/backend-go/internal/debt_payments"
	"github.com/Automaat/finance-buddy/backend-go/internal/debts"
	equitygrants "github.com/Automaat/finance-buddy/backend-go/internal/equity_grants"
	"github.com/Automaat/finance-buddy/backend-go/internal/exposure"
	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/goals"
	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
	"github.com/Automaat/finance-buddy/backend-go/internal/investment"
	"github.com/Automaat/finance-buddy/backend-go/internal/pit38"
	"github.com/Automaat/finance-buddy/backend-go/internal/quotes"
	"github.com/Automaat/finance-buddy/backend-go/internal/recurring"
	"github.com/Automaat/finance-buddy/backend-go/internal/retirement"
	"github.com/Automaat/finance-buddy/backend-go/internal/rules"
	"github.com/Automaat/finance-buddy/backend-go/internal/salaries"
	"github.com/Automaat/finance-buddy/backend-go/internal/scenarios"
	"github.com/Automaat/finance-buddy/backend-go/internal/simulations"
	"github.com/Automaat/finance-buddy/backend-go/internal/snapshots"
	"github.com/Automaat/finance-buddy/backend-go/internal/transactions"
	"github.com/Automaat/finance-buddy/backend-go/internal/zus"
)

// requestTimeout bounds every request via middleware.Timeout. Set above the
// longest legitimate handler (the 2-min quotes-refresh self-bound) so it acts
// as a backstop against stuck requests without cutting off real work.
const requestTimeout = 150 * time.Second

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
	// StooqAPIKey unlocks the Stooq daily-history endpoint (FB_STOOQ_APIKEY).
	// Empty means the scheduler/refresh handler falls back to the keyless
	// "latest" snapshot — daily backfill is then unavailable.
	StooqAPIKey string
	// FREDAPIKey selects FRED (OECD-sourced GUS CPI) as the monthly CPI
	// source. Empty falls back to Eurostat HICP (free, no key, but drifts
	// 0.1-0.3pp).
	FREDAPIKey string
}

// PickMonthlyCPIFetcher returns the (fetcher, sourceTag) the scheduler and
// /api/cpi/refresh-monthly should use. FRED if a key is configured, else
// Eurostat. Centralized here so server-side wiring and the scheduler stay
// in sync without duplicating the env-key check.
func PickMonthlyCPIFetcher(fredAPIKey string) (cpi.MonthlyFetcher, string) {
	if fred := cpi.NewFREDFetcher(fredAPIKey); fred != nil {
		return fred, cpi.FREDMonthlySource
	}
	return cpi.NewEurostatHICPFetcher(), cpi.EurostatMonthlySource
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
	// Per-request deadline. middleware.Timeout cancels r.Context() after
	// requestTimeout, which context-aware work (pgx queries, outbound scrapes)
	// observes and unwinds; it only writes a 504 if the handler returns past the
	// deadline without having already written. So this is a cooperative backstop,
	// not a hard kill of work that ignores context. Sized above the 2-min
	// self-bound /api/holdings/refresh-quotes pass, and below main.go's
	// WriteTimeout so the connection write deadline doesn't pre-empt it.
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitOrigins(cfg.CORSOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	// Pass a nil pinger (not a typed-nil *pgxpool.Pool) when there's no pool,
	// so the health probe's nil check works and avoids a panic on Ping.
	var healthPool pinger
	if deps.Pool != nil {
		healthPool = deps.Pool
	}
	r.Get("/health", healthHandler(logger, healthPool))

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
		registerAPIRoutes(r, cfg, pool, logger)
	})
}

func registerAPIRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	registerCoreRoutes(r, pool, logger)
	registerEquityRoutes(r, pool, logger)
	registerCPIAndPayrollRoutes(r, cfg, pool, logger)
	registerPortfolioRoutes(r, pool, logger)
	registerLedgerRoutes(r, cfg, pool, logger)
	registerDashboardRoutes(r, pool, logger)
}

func registerLedgerRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	txStore := transactions.NewStore(pool)
	txHandler := transactions.NewHandler(txStore, logger)
	r.Get("/api/accounts/{account_id}/transactions", txHandler.ListForAccount)
	r.Post("/api/accounts/{account_id}/transactions", txHandler.Create)
	r.Delete("/api/accounts/{account_id}/transactions/{transaction_id}", txHandler.Delete)
	r.Get("/api/transactions", txHandler.ListAll)
	r.Get("/api/transactions/counts", txHandler.Counts)
	r.Get("/api/transactions/types", txHandler.Types)

	recStore := recurring.NewStore(pool)
	recHandler := recurring.NewHandler(recStore, logger)
	r.Get("/api/recurring", recHandler.List)
	r.Post("/api/recurring", recHandler.Create)
	r.Get("/api/recurring/{id}", recHandler.Get)
	r.Put("/api/recurring/{id}", recHandler.Update)
	r.Delete("/api/recurring/{id}", recHandler.Delete)
	r.Post("/api/recurring/{id}/run-now", recHandler.RunNow)
	r.Post("/api/recurring/{id}/skip", recHandler.Skip)
	r.Post("/api/recurring/{id}/unskip", recHandler.Unskip)

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
	r.Get("/api/investment/returns", invHandler.Returns)

	registerHoldingsRoutes(r, cfg.StooqAPIKey, pool, logger)

	registerPIT38Routes(r, pool, logger)

	simHandler := simulations.NewHandler(simulations.NewStore(pool), logger)
	r.Post("/api/simulations/mortgage-vs-invest", simHandler.MortgageVsInvest)
	r.Post("/api/simulations/wibor", simHandler.WiborScenarios)
	r.Post("/api/simulations/retirement", simHandler.Retirement)
	r.Get("/api/simulations/prefill", simHandler.Prefill)
	r.Post("/api/simulations/monte-carlo", simHandler.MonteCarlo)

	scHandler := scenarios.NewHandler(scenarios.NewStore(pool), logger)
	r.Get("/api/scenarios", scHandler.List)
	r.Post("/api/scenarios", scHandler.Create)
	r.Get("/api/scenarios/{id}", scHandler.Get)
	r.Put("/api/scenarios/{id}", scHandler.Update)
	r.Delete("/api/scenarios/{id}", scHandler.Delete)
	r.Post("/api/scenarios/{id}/clone", scHandler.Clone)

	rulesHandler := rules.NewHandler(logger)
	r.Get("/api/rules", rulesHandler.List)
}

func registerDashboardRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	dashHandler := dashboard.NewHandler(dashboard.NewStore(pool), logger)
	r.Get("/api/dashboard", dashHandler.Get)

	expValuator := holdings.NewValuator(holdings.NewStore(pool), fx.NewService(pool, logger))
	expHandler := exposure.NewHandler(exposure.NewStore(pool), expValuator, logger)
	r.Get("/api/exposure/currency", expHandler.Currency)
}

func registerPIT38Routes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	pitHandler := pit38.NewHandler(pit38.NewStore(pool), fx.NewService(pool, logger), logger)
	r.Get("/api/pit38/realized", pitHandler.Realized)
}

func registerHoldingsRoutes(r chi.Router, stooqAPIKey string, pool *pgxpool.Pool, logger *slog.Logger) {
	hStore := holdings.NewStore(pool)
	hHandler := holdings.NewHandler(hStore, fx.NewService(pool, logger), logger)
	r.Get("/api/holdings", hHandler.Holdings)
	r.Get("/api/holdings/securities", hHandler.ListSecurities)
	r.Post("/api/holdings/securities", hHandler.CreateSecurity)
	r.Delete("/api/holdings/securities/{id}", hHandler.DeleteSecurity)
	r.Get("/api/holdings/securities/{id}/quotes", hHandler.ListQuotes)
	r.Post("/api/holdings/securities/{id}/quotes", hHandler.UpsertQuote)
	r.Get("/api/holdings/lots", hHandler.ListLots)
	r.Post("/api/holdings/lots", hHandler.CreateLot)
	r.Delete("/api/holdings/lots/{id}", hHandler.DeleteLot)

	stooq := quotes.NewStooqFetcher(stooqAPIKey)
	refresh := quotes.NewRefreshHandler(hStore, stooq, logger)
	r.Post("/api/holdings/refresh-quotes", refresh.Refresh)
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

func registerCPIAndPayrollRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	cpiStore := cpi.NewStore(pool)
	monthlyFetcher, monthlySource := PickMonthlyCPIFetcher(cfg.FREDAPIKey)
	cpiHandler := cpi.NewHandler(cpiStore, cpi.NewGUSFetcher(), monthlyFetcher, monthlySource, logger)
	r.Route("/api/cpi", func(r chi.Router) {
		r.Get("/series", cpiHandler.GetSeries)
		r.Post("/adjust", cpiHandler.Adjust)
		r.Post("/refresh", cpiHandler.Refresh)
		r.Post("/refresh-monthly", cpiHandler.RefreshMonthly)
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
	holdingsValuator := holdings.NewValuator(holdings.NewStore(pool), fx.NewService(pool, logger))
	bondsValuator := bonds.NewValuator(bonds.NewStore(pool), cpi.NewStore(pool), logger)
	accountsHandler := accounts.NewHandler(
		accounts.NewStore(pool, aggregatesStore), cpi.NewStore(pool), logger,
		holdingsValuator, bondsValuator,
	)
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

	bondsHandler := bonds.NewHandler(
		bonds.NewStore(pool), cpi.NewStore(pool),
		bondrates.NewObligacjeSkarbowePLFetcher(), logger,
	)
	r.Route("/api/bonds", func(r chi.Router) {
		r.Get("/", bondsHandler.List)
		r.Post("/", bondsHandler.Create)
		r.Get("/lookup", bondsHandler.Lookup)
		r.Get("/maturity-ladder", bondsHandler.MaturityLadder)
		r.Get("/{id}", bondsHandler.Get)
		r.Get("/{id}/ytm", bondsHandler.YTM)
		r.Put("/{id}", bondsHandler.Update)
		r.Delete("/{id}", bondsHandler.Delete)
	})

	allocStore := allocation.NewStore(pool)
	allocHoldings := allocation.NewHoldingsFromSnapshots(pool)
	allocHandler := allocation.NewHandler(allocStore, allocHoldings, logger)
	r.Route("/api/allocation", func(r chi.Router) {
		r.Get("/targets", allocHandler.List)
		r.Post("/targets", allocHandler.Create)
		r.Put("/targets/replace", allocHandler.Replace)
		r.Put("/targets/{id}", allocHandler.Update)
		r.Delete("/targets/{id}", allocHandler.Delete)
		r.Get("/drift", allocHandler.Drift)
	})
}

// pinger is the subset of *pgxpool.Pool the health probe needs. Narrowed to an
// interface so /health-only tests can pass a nil pool (skips the DB probe).
type pinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(logger *slog.Logger, pool pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A nil pool means the router was built without DB deps (health-only
		// tests); report ok so those keep passing. With a real pool, a failed
		// ping means Traefik/Docker should treat the backend as unready.
		if pool != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				logger.Error("health ping failed", "err", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				if encErr := json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"}); encErr != nil {
					logger.Error("encode health response", "err", encErr)
				}
				return
			}
		}
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
