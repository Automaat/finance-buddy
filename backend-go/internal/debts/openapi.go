package debts

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/debts", Tag: "debts", Summary: "List debts", Response: listResponse{}},
	{Method: "POST", Path: "/api/accounts/{account_id}/debts", Tag: "debts", Summary: "Create a debt", Response: response{}, Status: 201},
	{Method: "GET", Path: "/api/debts/{id}", Tag: "debts", Summary: "Get a debt", Response: response{}},
	{Method: "PUT", Path: "/api/debts/{id}", Tag: "debts", Summary: "Update a debt", Response: response{}},
	{Method: "DELETE", Path: "/api/debts/{id}", Tag: "debts", Summary: "Delete a debt", Status: 204},
}
