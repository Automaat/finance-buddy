package equitygrants

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/equity-grants", Tag: "equity-grants", Summary: "List equity grants", Response: listResponse{}},
	{Method: "GET", Path: "/api/equity-grants/{id}", Tag: "equity-grants", Summary: "Get an equity grant", Response: response{}},
	{Method: "POST", Path: "/api/equity-grants", Tag: "equity-grants", Summary: "Create an equity grant", Response: response{}, Status: 201},
	{Method: "PATCH", Path: "/api/equity-grants/{id}", Tag: "equity-grants", Summary: "Update an equity grant", Response: response{}},
	{Method: "DELETE", Path: "/api/equity-grants/{id}", Tag: "equity-grants", Summary: "Delete an equity grant", Status: 204},
}
