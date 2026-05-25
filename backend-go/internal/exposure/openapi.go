package exposure

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/exposure/currency", Tag: "exposure",
		Summary: "Currency exposure for the latest snapshot",
		Query: []apispec.QueryParam{
			{
				Name:        "target_pln_pct",
				Type:        "number",
				Description: "Optional PLN target percentage; enables drift reporting when present.",
			},
			{
				Name:        "tolerance",
				Type:        "number",
				Description: "Optional non-negative drift tolerance percentage; defaults to 5.",
			},
		},
		Response: Report{},
	},
}
