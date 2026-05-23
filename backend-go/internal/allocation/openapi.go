package allocation

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/allocation/targets", Tag: "allocation", Summary: "List allocation targets", Response: listResponse{}},
	{Method: "POST", Path: "/api/allocation/targets", Tag: "allocation", Summary: "Create allocation target", Request: createRequest{}, Response: response{}, Status: 201},
	{Method: "PUT", Path: "/api/allocation/targets/replace", Tag: "allocation", Summary: "Replace all targets for one owner scope", Request: replaceRequest{}, Response: listResponse{}},
	{Method: "PUT", Path: "/api/allocation/targets/{id}", Tag: "allocation", Summary: "Update allocation target", Request: updateRequest{}, Response: response{}},
	{Method: "DELETE", Path: "/api/allocation/targets/{id}", Tag: "allocation", Summary: "Delete allocation target", Status: 204},
	{Method: "GET", Path: "/api/allocation/drift", Tag: "allocation", Summary: "Actual vs target allocation drift", Response: driftResponse{}},
}
