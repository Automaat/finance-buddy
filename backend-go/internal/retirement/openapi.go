package retirement

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{Method: "GET", Path: "/api/retirement/stats", Tag: "retirement", Summary: "IKE/IKZE yearly stats", Response: []yearlyStat{}},
	{Method: "GET", Path: "/api/retirement/ppk-stats", Tag: "retirement", Summary: "PPK stats per owner", Response: []ppkStat{}},
	{Method: "POST", Path: "/api/retirement/ppk-contributions/generate", Tag: "retirement", Summary: "Generate monthly PPK contributions", Response: ppkGenerateResponse{}},
	{Method: "GET", Path: "/api/retirement/limits/{year}", Tag: "retirement", Summary: "Contribution limits for a year", Response: []limitResponse{}},
	{Method: "PUT", Path: "/api/retirement/limits/{year}/{wrapper}/{owner}", Tag: "retirement", Summary: "Upsert a contribution limit", Response: limitResponse{}},
}
