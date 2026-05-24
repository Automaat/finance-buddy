package main

import (
	"github.com/Automaat/finance-buddy/backend-go/internal/accounts"
	"github.com/Automaat/finance-buddy/backend-go/internal/allocation"
	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
	"github.com/Automaat/finance-buddy/backend-go/internal/assets"
	"github.com/Automaat/finance-buddy/backend-go/internal/bonds"
	bonusevents "github.com/Automaat/finance-buddy/backend-go/internal/bonus_events"
	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/config"
	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/dashboard"
	debtpayments "github.com/Automaat/finance-buddy/backend-go/internal/debt_payments"
	"github.com/Automaat/finance-buddy/backend-go/internal/debts"
	equitygrants "github.com/Automaat/finance-buddy/backend-go/internal/equity_grants"
	"github.com/Automaat/finance-buddy/backend-go/internal/goals"
	"github.com/Automaat/finance-buddy/backend-go/internal/holdings"
	"github.com/Automaat/finance-buddy/backend-go/internal/investment"
	"github.com/Automaat/finance-buddy/backend-go/internal/recurring"
	"github.com/Automaat/finance-buddy/backend-go/internal/retirement"
	"github.com/Automaat/finance-buddy/backend-go/internal/salaries"
	"github.com/Automaat/finance-buddy/backend-go/internal/scenarios"
	"github.com/Automaat/finance-buddy/backend-go/internal/simulations"
	"github.com/Automaat/finance-buddy/backend-go/internal/snapshots"
	"github.com/Automaat/finance-buddy/backend-go/internal/transactions"
	"github.com/Automaat/finance-buddy/backend-go/internal/zus"
)

// healthResponse mirrors the inline /health payload (it has no endpoint
// package of its own).
type healthResponse struct {
	Status string `json:"status"`
}

// allRoutes aggregates every endpoint package's APISpec into a single,
// path-sorted list. New endpoint packages are added here.
func allRoutes() []apispec.Route {
	routes := []apispec.Route{
		{
			Method: "GET", Path: "/health", Tag: "system",
			Summary:  "Liveness probe",
			Response: healthResponse{},
		},
	}
	specs := [][]apispec.Route{
		config.APISpec,
		goals.APISpec,
		accounts.APISpec,
		assets.APISpec,
		bonds.APISpec,
		snapshots.APISpec,
		dashboard.APISpec,
		companyvaluations.APISpec,
		bonusevents.APISpec,
		equitygrants.APISpec,
		cpi.APISpec,
		salaries.APISpec,
		zus.APISpec,
		retirement.APISpec,
		investment.APISpec,
		holdings.APISpec,
		simulations.APISpec,
		scenarios.APISpec,
		transactions.APISpec,
		recurring.APISpec,
		debtpayments.APISpec,
		debts.APISpec,
		allocation.APISpec,
	}
	for _, s := range specs {
		routes = append(routes, s...)
	}
	return sortedRoutes(routes)
}
