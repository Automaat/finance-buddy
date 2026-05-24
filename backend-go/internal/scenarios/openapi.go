package scenarios

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "GET", Path: "/api/scenarios", Tag: "scenarios",
		Summary:  "List saved simulation scenarios",
		Response: listResponse{},
	},
	{
		Method: "GET", Path: "/api/scenarios/{id}", Tag: "scenarios",
		Summary:  "Get a saved scenario",
		Response: response{},
	},
	{
		Method: "POST", Path: "/api/scenarios", Tag: "scenarios",
		Summary:  "Save a new scenario",
		Request:  createRequest{},
		Response: response{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/scenarios/{id}", Tag: "scenarios",
		Summary:  "Rename + replace inputs on a scenario",
		Request:  updateRequest{},
		Response: response{},
	},
	{
		Method: "POST", Path: "/api/scenarios/{id}/clone", Tag: "scenarios",
		Summary:  "Duplicate a scenario",
		Request:  cloneRequest{},
		Response: response{},
		Status:   201,
	},
	{
		Method: "DELETE", Path: "/api/scenarios/{id}", Tag: "scenarios",
		Summary: "Delete a scenario",
		Status:  204,
	},
}
