package personas

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/personas", Tag: "personas",
		Summary:  "List personas",
		Response: []response{},
	},
	{
		Method: "POST", Path: "/api/personas", Tag: "personas",
		Summary:  "Create a persona",
		Request:  createRequest{},
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/personas/{id}", Tag: "personas",
		Summary:  "Update a persona",
		Request:  updateRequest{},
		Response: response{},
	},
	{
		Method: "DELETE", Path: "/api/personas/{id}", Tag: "personas",
		Summary: "Delete a persona",
		Status:  204,
	},
}
