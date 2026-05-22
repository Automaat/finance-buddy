package snapshots

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/snapshots", Tag: "snapshots",
		Summary:  "List snapshots",
		Response: []response{},
	},
	{
		Method: "GET", Path: "/api/snapshots/{id}", Tag: "snapshots",
		Summary:  "Get a snapshot",
		Response: response{},
	},
	{
		Method: "POST", Path: "/api/snapshots", Tag: "snapshots",
		Summary:  "Create a snapshot",
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/snapshots/{id}", Tag: "snapshots",
		Summary:  "Update a snapshot",
		Response: response{},
	},
}
