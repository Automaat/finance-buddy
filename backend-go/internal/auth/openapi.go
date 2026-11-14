package auth

import "github.com/Automaat/finance-buddy/backend-go/internal/apispec"

// APISpec registers this package's routes for OpenAPI generation.
var APISpec = []apispec.Route{
	{
		Method: "POST", Path: "/api/auth/login", Tag: "auth",
		Summary:  "Log in",
		Request:  loginRequest{},
		Response: loginResponse{},
	},
	{
		Method: "POST", Path: "/api/auth/logout", Tag: "auth",
		Summary: "Log out",
		Status:  204,
	},
	{
		Method: "GET", Path: "/api/auth/oidc/status", Tag: "auth",
		Summary:  "Report whether OIDC login is enabled",
		Response: oidcStatusResponse{},
	},
	{
		Method: "GET", Path: "/api/auth/oidc/login", Tag: "auth",
		Summary: "Start OIDC login (redirects to the provider)",
		Status:  302,
	},
	{
		Method: "GET", Path: "/api/auth/oidc/callback", Tag: "auth",
		Summary: "Finish OIDC login (redirects to the app)",
		Status:  302,
	},
	{
		Method: "GET", Path: "/api/auth/me", Tag: "auth",
		Summary:  "Read current user",
		Response: userResponse{},
	},
	{
		Method: "GET", Path: "/api/users", Tag: "auth",
		Summary:  "List owner options",
		Response: []ownerOption{},
	},
	{
		Method: "GET", Path: "/api/auth/users", Tag: "auth",
		Summary:  "List users",
		Response: []userResponse{},
	},
	{
		Method: "POST", Path: "/api/auth/users", Tag: "auth",
		Summary:  "Create a user",
		Request:  createUserRequest{},
		Response: userResponse{},
		Status:   201,
	},
	{
		Method: "PUT", Path: "/api/auth/users/{id}", Tag: "auth",
		Summary:  "Update a user",
		Request:  updateUserRequest{},
		Response: userResponse{},
	},
}
