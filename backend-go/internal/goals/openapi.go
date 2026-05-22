package goals

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/goals", Tag: "goals",
		Summary:  "List goals",
		Response: listResponse{},
	},
	{
		Method: "GET", Path: "/api/goals/{id}", Tag: "goals",
		Summary:  "Get a goal",
		Response: response{},
	},
	{
		Method: "POST", Path: "/api/goals", Tag: "goals",
		Summary:  "Create a goal",
		Request:  createRequest{},
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/goals/{id}", Tag: "goals",
		Summary:  "Update a goal",
		Response: response{},
	},
	{
		Method: "DELETE", Path: "/api/goals/{id}", Tag: "goals",
		Summary: "Delete a goal",
		Status:  204,
	},
}
