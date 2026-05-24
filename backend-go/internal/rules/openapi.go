package rules

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/rules", Tag: "rules",
		Summary:  "List Polish financial constants with source metadata",
		Response: listResponse{},
	},
}
