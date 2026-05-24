package holdings

import (
	"net/http"

	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
)

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/holdings", Tag: "holdings", Summary: "Aggregated holdings across accounts", Response: holdingsResponse{}},
	{Method: "GET", Path: "/api/holdings/securities", Tag: "holdings", Summary: "List securities", Response: listSecuritiesResponse{}},
	{Method: "POST", Path: "/api/holdings/securities", Tag: "holdings", Summary: "Create a security", Status: http.StatusCreated, Response: securityResponse{}},
	{Method: "DELETE", Path: "/api/holdings/securities/{id}", Tag: "holdings", Summary: "Delete a security", Status: http.StatusNoContent},
	{Method: "GET", Path: "/api/holdings/securities/{id}/quotes", Tag: "holdings", Summary: "List manual price quotes", Response: listQuotesResponse{}},
	{Method: "POST", Path: "/api/holdings/securities/{id}/quotes", Tag: "holdings", Summary: "Upsert a manual price quote", Response: quoteResponse{}},
	{Method: "GET", Path: "/api/holdings/lots", Tag: "holdings", Summary: "List lots (filterable by account_id/security_id)", Response: listLotsResponse{}},
	{Method: "POST", Path: "/api/holdings/lots", Tag: "holdings", Summary: "Create a lot", Status: http.StatusCreated, Response: lotResponse{}},
	{Method: "DELETE", Path: "/api/holdings/lots/{id}", Tag: "holdings", Summary: "Delete a lot", Status: http.StatusNoContent},
}
