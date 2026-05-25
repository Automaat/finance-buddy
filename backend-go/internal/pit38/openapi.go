package pit38

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/pit38/realized", Tag: "pit38",
		Summary: "PIT-38 realized gains worksheet",
		Query: []apispec.QueryParam{
			{
				Name:        "year",
				Type:        "integer",
				Description: "Optional report year; defaults from PIT filing window.",
			},
			{
				Name:        "format",
				Type:        "string",
				Enum:        []string{"csv"},
				Description: "Optional csv value returns a downloadable text/csv report.",
			},
		},
		Response: Report{},
	},
}
