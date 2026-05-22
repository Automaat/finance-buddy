package cpi

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/cpi/series", Tag: "cpi", Summary: "Get the CPI year-over-year series", Response: seriesResponse{}},
	{Method: "POST", Path: "/api/cpi/adjust", Tag: "cpi", Summary: "Inflation-adjust an amount between two dates", Response: adjustResponse{}},
	{Method: "POST", Path: "/api/cpi/refresh", Tag: "cpi", Summary: "Refresh CPI data from GUS", Response: refreshResponse{}},
}
