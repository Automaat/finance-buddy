package main

import (
	"github.com/Automaat/finance-buddy/backend-go/internal/accounts"
	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
	"github.com/Automaat/finance-buddy/backend-go/internal/assets"
	"github.com/Automaat/finance-buddy/backend-go/internal/config"
	"github.com/Automaat/finance-buddy/backend-go/internal/dashboard"
	"github.com/Automaat/finance-buddy/backend-go/internal/goals"
	"github.com/Automaat/finance-buddy/backend-go/internal/personas"
	"github.com/Automaat/finance-buddy/backend-go/internal/snapshots"
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
	routes = append(routes, config.APISpec...)
	routes = append(routes, personas.APISpec...)
	routes = append(routes, goals.APISpec...)
	routes = append(routes, accounts.APISpec...)
	routes = append(routes, assets.APISpec...)
	routes = append(routes, snapshots.APISpec...)
	routes = append(routes, dashboard.APISpec...)
	return sortedRoutes(routes)
}
