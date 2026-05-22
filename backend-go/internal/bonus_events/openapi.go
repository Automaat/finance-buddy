package bonusevents

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/bonuses", Tag: "bonuses", Summary: "List bonus events", Response: listResponse{}},
	{Method: "GET", Path: "/api/bonuses/{id}", Tag: "bonuses", Summary: "Get a bonus event", Response: response{}},
	{Method: "POST", Path: "/api/bonuses", Tag: "bonuses", Summary: "Create a bonus event", Response: response{}, Status: 201},
	{Method: "PATCH", Path: "/api/bonuses/{id}", Tag: "bonuses", Summary: "Update a bonus event", Response: response{}},
	{Method: "DELETE", Path: "/api/bonuses/{id}", Tag: "bonuses", Summary: "Delete a bonus event", Status: 204},
}
