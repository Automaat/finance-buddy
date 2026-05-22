// Command gen-openapi reflects the route registry into an OpenAPI 3.0 spec.
//
// Every endpoint-group package exports `APISpec []apispec.Route`; this command
// aggregates them, reflects each request/response Go struct's JSON tags into a
// schema, and writes api/openapi-go.json. CI re-runs it and fails on drift.
//
// To add an endpoint to the spec, register it in the owning package's APISpec
// — see backend-go/README.md.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"

	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen-openapi:", err)
		os.Exit(1)
	}
}

func run() error {
	routes := allRoutes()
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Finance Buddy API",
			Description: "Generated from backend-go route registry — do not edit by hand.",
			Version:     "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}
	gen := openapi3gen.NewGenerator(
		openapi3gen.UseAllExportedFields(),
		openapi3gen.SchemaCustomizer(customizeScalar),
	)
	for i := range routes {
		if err := addRoute(doc, gen, &routes[i]); err != nil {
			return fmt.Errorf("%s %s: %w", routes[i].Method, routes[i].Path, err)
		}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}
	out = append(out, '\n')
	dest := specPath()
	if err := os.WriteFile(dest, out, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	fmt.Printf("wrote %s — %d routes\n", dest, len(routes))
	return nil
}

// specPath resolves <repo>/api/openapi-go.json relative to this source file's
// module root, so the command works regardless of the caller's CWD.
func specPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return "api/openapi-go.json"
	}
	// Walk up to the repo root (the dir containing backend-go/).
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "backend-go")); err == nil {
			return filepath.Join(dir, "api", "openapi-go.json")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(wd, "api", "openapi-go.json")
		}
		dir = parent
	}
}

func addRoute(doc *openapi3.T, gen *openapi3gen.Generator, rt *apispec.Route) error {
	op := &openapi3.Operation{
		Summary: rt.Summary,
		Tags:    []string{rt.Tag},
	}
	for _, p := range pathParams(rt.Path) {
		op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     p,
				In:       "path",
				Required: true,
				Schema:   openapi3.NewStringSchema().NewRef(),
			},
		})
	}
	if rt.Request != nil {
		schema, err := gen.NewSchemaRefForValue(rt.Request, nil)
		if err != nil {
			return fmt.Errorf("request schema: %w", err)
		}
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: openapi3.NewRequestBody().
				WithRequired(true).
				WithJSONSchemaRef(schema),
		}
	}
	status := rt.Status
	if status == 0 {
		status = 200
	}
	resp := openapi3.NewResponse().WithDescription(statusText(status))
	if rt.Response != nil {
		schema, err := gen.NewSchemaRefForValue(rt.Response, nil)
		if err != nil {
			return fmt.Errorf("response schema: %w", err)
		}
		resp = resp.WithJSONSchemaRef(schema)
	}
	op.AddResponse(status, resp)

	item := doc.Paths.Find(rt.Path)
	if item == nil {
		item = &openapi3.PathItem{}
		doc.Paths.Set(rt.Path, item)
	}
	switch rt.Method {
	case http.MethodGet:
		item.Get = op
	case http.MethodPost:
		item.Post = op
	case http.MethodPut:
		item.Put = op
	case http.MethodPatch:
		item.Patch = op
	case http.MethodDelete:
		item.Delete = op
	default:
		return fmt.Errorf("unsupported method %q", rt.Method)
	}
	return nil
}

// customizeScalar overrides the schema for the wire scalar-wrapper types
// (pyFloat, isoDate, isoNaive, moneyJSON, ppkRate). Without this, openapi3gen
// reflects e.g. moneyJSON (underlying decimal.Decimal, a struct) into a
// meaningless object schema. Match is by type name — the wrappers follow a
// consistent naming convention across every endpoint package.
func customizeScalar(_ string, t reflect.Type, _ reflect.StructTag, schema *openapi3.Schema) error {
	switch t.Name() {
	case "pyFloat", "moneyJSON", "ppkRate":
		schema.Type = &openapi3.Types{"number"}
		schema.Properties = nil
	case "isoDate":
		schema.Type = &openapi3.Types{"string"}
		schema.Format = "date"
		schema.Properties = nil
	case "isoNaive":
		schema.Type = &openapi3.Types{"string"}
		schema.Format = "date-time"
		schema.Properties = nil
	}
	return nil
}

func pathParams(path string) []string {
	var out []string
	for seg := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, seg[1:len(seg)-1])
		}
	}
	return out
}

func statusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	default:
		return fmt.Sprintf("status %d", code)
	}
}

// sortedRoutes keeps the emitted spec deterministic regardless of package
// aggregation order.
func sortedRoutes(routes []apispec.Route) []apispec.Route {
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}
