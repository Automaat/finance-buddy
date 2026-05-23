package simulations

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
//
// The retirement + prefill responses are assembled as dynamic maps in the
// handler, so only their paths are registered (no response schema).
var APISpec = []apispec.Route{
	{Method: "POST", Path: "/api/simulations/mortgage-vs-invest", Tag: "simulations", Summary: "Mortgage overpay vs invest comparison", Response: mortgageResponse{}},
	{Method: "POST", Path: "/api/simulations/retirement", Tag: "simulations", Summary: "Retirement account projection"},
	{Method: "GET", Path: "/api/simulations/prefill", Tag: "simulations", Summary: "Prefill simulation inputs"},
	{Method: "POST", Path: "/api/simulations/monte-carlo", Tag: "simulations", Summary: "Monte Carlo retirement projection", Response: MonteCarloResult{}},
}
