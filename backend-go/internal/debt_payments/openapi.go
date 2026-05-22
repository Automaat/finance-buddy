package debtpayments

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/accounts/{account_id}/payments", Tag: "debt-payments", Summary: "List an account's debt payments", Response: listResponse{}},
	{Method: "POST", Path: "/api/accounts/{account_id}/payments", Tag: "debt-payments", Summary: "Create a debt payment", Request: createRequest{}, Response: response{}, Status: 201},
	{Method: "DELETE", Path: "/api/accounts/{account_id}/payments/{payment_id}", Tag: "debt-payments", Summary: "Delete a debt payment", Status: 204},
	{Method: "GET", Path: "/api/payments", Tag: "debt-payments", Summary: "List all debt payments", Response: listResponse{}},
	{Method: "GET", Path: "/api/payments/counts", Tag: "debt-payments", Summary: "Debt-payment counts per account", Response: map[string]int{}},
}
