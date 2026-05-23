package bonds

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/bonds", Tag: "bonds", Summary: "List treasury bonds", Response: listResponse{}},
	{Method: "GET", Path: "/api/bonds/{id}", Tag: "bonds", Summary: "Get a treasury bond", Response: response{}},
	{Method: "GET", Path: "/api/bonds/{id}/ytm", Tag: "bonds", Summary: "Yield-to-maturity projection", Response: ytmResponse{}},
	{Method: "POST", Path: "/api/bonds", Tag: "bonds", Summary: "Create a treasury bond", Response: response{}, Status: 201},
	{Method: "PUT", Path: "/api/bonds/{id}", Tag: "bonds", Summary: "Update a treasury bond", Response: response{}},
	{Method: "DELETE", Path: "/api/bonds/{id}", Tag: "bonds", Summary: "Delete a treasury bond", Status: 204},
}
