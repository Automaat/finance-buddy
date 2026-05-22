package transactions

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/accounts/{account_id}/transactions", Tag: "transactions", Summary: "List an account's transactions", Response: listResponse{}},
	{Method: "POST", Path: "/api/accounts/{account_id}/transactions", Tag: "transactions", Summary: "Create a transaction", Request: createRequest{}, Response: response{}, Status: 201},
	{Method: "DELETE", Path: "/api/accounts/{account_id}/transactions/{transaction_id}", Tag: "transactions", Summary: "Delete a transaction", Status: 204},
	{Method: "GET", Path: "/api/transactions", Tag: "transactions", Summary: "List all transactions", Response: listResponse{}},
	{Method: "GET", Path: "/api/transactions/counts", Tag: "transactions", Summary: "Transaction counts per account", Response: map[string]int{}},
}
