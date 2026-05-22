package companyvaluations

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/company-valuations", Tag: "company-valuations", Summary: "List company valuations", Response: listResponse{}},
	{Method: "GET", Path: "/api/company-valuations/{id}", Tag: "company-valuations", Summary: "Get a company valuation", Response: response{}},
	{Method: "POST", Path: "/api/company-valuations", Tag: "company-valuations", Summary: "Create a company valuation", Response: response{}, Status: 201},
	{Method: "PATCH", Path: "/api/company-valuations/{id}", Tag: "company-valuations", Summary: "Update a company valuation", Response: response{}},
	{Method: "DELETE", Path: "/api/company-valuations/{id}", Tag: "company-valuations", Summary: "Delete a company valuation", Status: 204},
}
