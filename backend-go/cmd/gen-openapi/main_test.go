package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"

	"github.com/Automaat/finance-buddy/backend-go/internal/apispec"
)

func TestAddRouteIncludesQueryParams(t *testing.T) {
	doc := &openapi3.T{Paths: openapi3.NewPaths()}
	gen := openapi3gen.NewGenerator()
	rt := apispec.Route{
		Method:  http.MethodGet,
		Path:    "/api/reports/{year}",
		Tag:     "reports",
		Summary: "Read report",
		Query: []apispec.QueryParam{
			{
				Name:        "format",
				Type:        "string",
				Description: "Download format",
				Enum:        []string{"csv"},
				Required:    true,
			},
			{
				Name:   "target",
				Type:   "number",
				Format: "double",
			},
		},
	}

	if err := addRoute(doc, gen, &rt); err != nil {
		t.Fatalf("add route: %v", err)
	}

	item := doc.Paths.Find(rt.Path)
	if item == nil {
		t.Fatalf("path item %q missing", rt.Path)
	}
	op := item.Get
	if op == nil {
		t.Fatal("GET operation missing")
	}

	year := findParam(t, op.Parameters, "year", "path")
	if !year.Required {
		t.Fatal("year path param is not required")
	}
	requireSchemaType(t, year.Schema.Value, "integer")

	format := findParam(t, op.Parameters, "format", "query")
	if !format.Required {
		t.Fatal("format query param is not required")
	}
	if format.Description != "Download format" {
		t.Fatalf("format description = %q", format.Description)
	}
	requireSchemaType(t, format.Schema.Value, "string")
	if len(format.Schema.Value.Enum) != 1 || format.Schema.Value.Enum[0] != "csv" {
		t.Fatalf("format enum = %#v, want [csv]", format.Schema.Value.Enum)
	}

	target := findParam(t, op.Parameters, "target", "query")
	if target.Required {
		t.Fatal("target query param is required")
	}
	requireSchemaType(t, target.Schema.Value, "number")
	if target.Schema.Value.Format != "double" {
		t.Fatalf("target format = %q, want double", target.Schema.Value.Format)
	}
}

func TestQueryParamSchemaTypes(t *testing.T) {
	tests := []struct {
		name string
		p    apispec.QueryParam
		want string
	}{
		{
			name: "default string",
			p:    apispec.QueryParam{Name: "q"},
			want: "string",
		},
		{
			name: "integer",
			p:    apispec.QueryParam{Name: "year", Type: "integer"},
			want: "integer",
		},
		{
			name: "number",
			p:    apispec.QueryParam{Name: "pct", Type: "number"},
			want: "number",
		},
		{
			name: "boolean",
			p:    apispec.QueryParam{Name: "active", Type: "boolean"},
			want: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := queryParamSchema(tt.p).Value
			requireSchemaType(t, schema, tt.want)
		})
	}
}

func TestAddRouteRejectsUnsupportedMethod(t *testing.T) {
	doc := &openapi3.T{Paths: openapi3.NewPaths()}
	gen := openapi3gen.NewGenerator()
	rt := apispec.Route{
		Method:  http.MethodTrace,
		Path:    "/api/reports",
		Tag:     "reports",
		Summary: "Trace report",
	}

	err := addRoute(doc, gen, &rt)
	if err == nil {
		t.Fatal("add route succeeded")
	}
	if !strings.Contains(err.Error(), `unsupported method "TRACE"`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestAllRoutesIncludesAuthRoutes(t *testing.T) {
	got := routeSet(allRoutes())
	for _, key := range []string{
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"GET /api/auth/me",
		"GET /api/users",
		"GET /api/auth/users",
		"POST /api/auth/users",
		"PUT /api/auth/users/{id}",
	} {
		if !got[key] {
			t.Fatalf("%s missing from allRoutes", key)
		}
	}
}

func routeSet(routes []apispec.Route) map[string]bool {
	out := make(map[string]bool, len(routes))
	for i := range routes {
		out[routes[i].Method+" "+routes[i].Path] = true
	}
	return out
}

func findParam(
	t *testing.T,
	params openapi3.Parameters,
	name string,
	location string,
) *openapi3.Parameter {
	t.Helper()
	for _, param := range params {
		if param.Value.Name == name && param.Value.In == location {
			return param.Value
		}
	}
	t.Fatalf("%s %s param missing", location, name)
	return nil
}

func requireSchemaType(t *testing.T, schema *openapi3.Schema, want string) {
	t.Helper()
	if schema.Type == nil || !schema.Type.Is(want) {
		t.Fatalf("schema type = %#v, want %s", schema.Type, want)
	}
}
