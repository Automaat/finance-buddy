package accounts

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/accounts", Tag: "accounts",
		Summary:  "List accounts",
		Response: listResponse{},
	},
	{
		Method: "POST", Path: "/api/accounts", Tag: "accounts",
		Summary:  "Create an account",
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/accounts/{id}", Tag: "accounts",
		Summary:  "Update an account",
		Response: response{},
	},
	{
		Method: "DELETE", Path: "/api/accounts/{id}", Tag: "accounts",
		Summary: "Delete an account",
		Status:  204,
	},
}
