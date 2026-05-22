package investment

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/investment/stock-stats", Tag: "investment", Summary: "Stock ROI aggregates", Response: response{}},
	{Method: "GET", Path: "/api/investment/bond-stats", Tag: "investment", Summary: "Bond ROI aggregates", Response: response{}},
}
