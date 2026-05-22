package salaries

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/salaries", Tag: "salaries", Summary: "List salary records", Response: listResponse{}},
	{Method: "GET", Path: "/api/salaries/{id}", Tag: "salaries", Summary: "Get a salary record", Response: response{}},
	{Method: "POST", Path: "/api/salaries", Tag: "salaries", Summary: "Create a salary record", Response: response{}, Status: 201},
	{Method: "PATCH", Path: "/api/salaries/{id}", Tag: "salaries", Summary: "Update a salary record", Response: response{}},
	{Method: "DELETE", Path: "/api/salaries/{id}", Tag: "salaries", Summary: "Delete a salary record", Status: 204},
}
