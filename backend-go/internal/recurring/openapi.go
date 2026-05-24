package recurring

import (
	"net/http"

	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
)

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/recurring", Tag: "recurring", Summary: "List recurring templates", Response: listResponse{}},
	{Method: "POST", Path: "/api/recurring", Tag: "recurring", Summary: "Create a recurring template", Status: http.StatusCreated, Response: response{}},
	{Method: "GET", Path: "/api/recurring/{id}", Tag: "recurring", Summary: "Get a recurring template", Response: response{}},
	{Method: "PUT", Path: "/api/recurring/{id}", Tag: "recurring", Summary: "Update a recurring template", Response: response{}},
	{Method: "DELETE", Path: "/api/recurring/{id}", Tag: "recurring", Summary: "Delete a recurring template", Status: http.StatusNoContent},
	{Method: "POST", Path: "/api/recurring/{id}/run-now", Tag: "recurring", Summary: "Mint an ad-hoc transaction now", Status: http.StatusCreated, Response: runNowResponse{}},
	{Method: "POST", Path: "/api/recurring/{id}/skip", Tag: "recurring", Summary: "Skip an occurrence", Response: response{}},
	{Method: "POST", Path: "/api/recurring/{id}/unskip", Tag: "recurring", Summary: "Resume a previously skipped occurrence", Response: response{}},
}
