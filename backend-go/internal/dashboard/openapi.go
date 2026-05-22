package dashboard

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/dashboard", Tag: "dashboard",
		Summary:  "Dashboard — net worth, allocation, time series, deltas",
		Response: dashboardWire{},
	},
}
