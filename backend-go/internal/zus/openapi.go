package zus

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "POST", Path: "/api/zus/calculate", Tag: "zus", Summary: "Project a ZUS pension", Response: calculateResponse{}},
	{Method: "GET", Path: "/api/zus/prefill", Tag: "zus", Summary: "Prefill ZUS inputs from salary history", Response: prefillResponse{}},
}
