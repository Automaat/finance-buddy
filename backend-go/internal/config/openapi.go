package config

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/config", Tag: "config",
		Summary:  "Get application configuration",
		Response: response{},
	},
	{
		Method: "PUT", Path: "/api/config", Tag: "config",
		Summary:  "Update application configuration",
		Request:  request{},
		Response: response{},
	},
}
