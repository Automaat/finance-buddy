package assets

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/assets", Tag: "assets",
		Summary:  "List assets",
		Response: listResponse{},
	},
	{
		Method: "POST", Path: "/api/assets", Tag: "assets",
		Summary:  "Create an asset",
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/assets/{id}", Tag: "assets",
		Summary:  "Update an asset",
		Response: response{},
	},
	{
		Method: "DELETE", Path: "/api/assets/{id}", Tag: "assets",
		Summary: "Delete an asset",
		Status:  204,
	},
}
