// Package apispec is the route registry that drives OpenAPI generation.
//
// Each endpoint-group package declares an exported `APISpec []apispec.Route`
// listing its routes plus the Go request/response wire types. cmd/gen-openapi
// aggregates every package's APISpec and reflects the types into an OpenAPI
// 3.0 document — see backend-go/README.md.
//
// To expose a new endpoint in the spec: add a Route to the owning package's
// APISpec with a zero value of its request/response wire struct.
package apispec

// Route describes one endpoint for the OpenAPI generator.
type Route struct {
	// Method is the uppercase HTTP method (GET, POST, PUT, PATCH, DELETE).
	Method string
	// Path is the full route path with chi placeholders, e.g.
	// "/api/accounts/{id}".
	Path string
	// Tag groups the endpoint in the spec (usually the domain name).
	Tag string
	// Summary is a one-line human description.
	Summary string
	// Request is a zero value of the request-body wire type, or nil when the
	// endpoint takes no body.
	Request any
	// Response is a zero value of the 200/201 response wire type, or nil when
	// the endpoint returns no body.
	Response any
	// Status is the success status code (defaults to 200 when zero).
	Status int
}
