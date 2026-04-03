package apidocs

import (
	"fmt"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/auth"
)

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type switchTenantRequest struct {
	RefreshToken string `json:"refresh_token"`
	TenantID     string `json:"tenant_id"`
	Password     string `json:"password"`
}

type createInviteRequest struct {
	Email string    `json:"email"`
	Role  auth.Role `json:"role"`
}

type manualNudgeResponse struct {
	Status  string `json:"status"`
	Student string `json:"student"`
	Channel string `json:"channel"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func Build() (*Document, error) {
	registry := newSchemaRegistry()

	doc := &Document{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       "P&AI Bot API",
			Version:     "0.1.0",
			Description: "OpenAPI reference for the stdlib Go server. Paths are explicit; component schemas are generated from Go request/response types.",
		},
		Servers: []Server{{URL: "/"}},
		Tags: []Tag{
			{Name: "Health"},
			{Name: "Auth"},
			{Name: "Admin"},
		},
		Components: Components{
			Schemas: map[string]*Schema{},
			SecuritySchemes: map[string]*SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
		Paths: Paths{},
	}

	doc.Paths["/healthz"] = route("GET", Operation{
		Summary:   "Health check",
		Tags:      []string{"Health"},
		Responses: okJSON("Service is healthy.", registry.refFor(healthResponse{})),
	})
	doc.Paths["/readyz"] = route("GET", Operation{
		Summary:   "Readiness check",
		Tags:      []string{"Health"},
		Responses: okJSON("Service is ready.", registry.refFor(healthResponse{})),
	})

	doc.Paths["/api/auth/login"] = route("POST", Operation{
		Summary:     "Issue access and refresh tokens",
		Tags:        []string{"Auth"},
		RequestBody: jsonBody(registry.refFor(auth.LoginRequest{})),
		Responses: mergeResponses(
			responseJSON("200", "Authentication succeeded.", registry.refFor(auth.TokenPair{})),
			responseMixed400(registry.refFor(tenantRequiredErrorResponse{})),
			responseText("401", "Credentials are invalid."),
			responseText("501", "Auth service is not implemented."),
		),
	})
	doc.Paths["/api/auth/invitations/accept"] = route("POST", Operation{
		Summary:     "Activate an invited account",
		Tags:        []string{"Auth"},
		RequestBody: jsonBody(registry.refFor(auth.AcceptInviteRequest{})),
		Responses: mergeResponses(
			responseJSON("201", "Invitation accepted.", registry.refFor(auth.TokenPair{})),
			responseText("400", "Request body is invalid."),
			responseText("401", "Invite token is invalid or expired."),
			responseText("501", "Auth service is not implemented."),
		),
	})
	doc.Paths["/api/auth/refresh"] = route("POST", Operation{
		Summary:     "Refresh an access token",
		Tags:        []string{"Auth"},
		RequestBody: jsonBody(registry.refFor(refreshTokenRequest{})),
		Responses: mergeResponses(
			responseJSON("200", "Token refresh succeeded.", registry.refFor(auth.TokenPair{})),
			responseText("400", "Request body is invalid."),
			responseText("401", "Refresh token is invalid."),
			responseText("501", "Auth service is not implemented."),
		),
	})
	doc.Paths["/api/auth/switch-tenant"] = route("POST", Operation{
		Summary:     "Switch the active tenant for a session",
		Tags:        []string{"Auth"},
		RequestBody: jsonBody(registry.refFor(switchTenantRequest{})),
		Responses: mergeResponses(
			responseJSON("200", "Tenant switch succeeded.", registry.refFor(auth.TokenPair{})),
			responseText("400", "Request body is invalid."),
			responseText("401", "Refresh token or password is invalid."),
			responseText("501", "Auth service is not implemented."),
		),
	})
	doc.Paths["/api/auth/logout"] = route("POST", Operation{
		Summary:     "Revoke a refresh token",
		Tags:        []string{"Auth"},
		RequestBody: jsonBody(registry.refFor(refreshTokenRequest{})),
		Responses: mergeResponses(
			responseEmpty("204", "Logout succeeded."),
			responseText("400", "Request body is invalid."),
			responseText("401", "Refresh token is invalid."),
			responseText("501", "Auth service is not implemented."),
		),
	})

	protected := []Security{{"BearerAuth": []string{}}}
	idParam := func(description string) []Parameter {
		return []Parameter{{
			Name:        "id",
			In:          "path",
			Required:    true,
			Description: description,
			Schema:      &Schema{Type: "string"},
		}}
	}

	doc.Paths["/api/admin/invites"] = route("POST", Operation{
		Summary:     "Create a teacher, parent, or admin invite",
		Tags:        []string{"Admin"},
		Security:    protected,
		RequestBody: jsonBody(registry.refFor(createInviteRequest{})),
		Responses: mergeResponses(
			responseJSON("201", "Invite created.", registry.refFor(auth.InviteRecord{})),
			responseText("400", "Request body is invalid."),
			responseText("401", "Request is not authenticated."),
			responseText("403", "Caller is not allowed to issue invites."),
			responseText("409", "Invite already exists."),
			responseText("501", "Auth service is not implemented."),
		),
	})
	doc.Paths["/api/admin/classes/{id}/progress"] = route("GET", Operation{
		Summary:    "Get class mastery progress",
		Tags:       []string{"Admin"},
		Security:   protected,
		Parameters: idParam("Class identifier."),
		Responses: mergeResponses(
			responseJSON("200", "Class progress summary.", registry.refFor(adminapi.ClassProgress{})),
			protectedErrors(),
			responseText("404", "Requested class data was not found."),
		),
	})
	doc.Paths["/api/admin/students/{id}"] = route("GET", Operation{
		Summary:    "Get student detail",
		Tags:       []string{"Admin"},
		Security:   protected,
		Parameters: idParam("Student identifier."),
		Responses: mergeResponses(
			responseJSON("200", "Student detail payload.", registry.refFor(adminapi.StudentDetail{})),
			protectedErrors(),
			responseText("404", "Requested student was not found."),
		),
	})
	doc.Paths["/api/admin/students/{id}/conversations"] = route("GET", Operation{
		Summary:    "Get student conversation history",
		Tags:       []string{"Admin"},
		Security:   protected,
		Parameters: idParam("Student identifier."),
		Responses: mergeResponses(
			responseJSON("200", "Conversation history.", arrayOf(registry.refFor(adminapi.StudentConversation{}))),
			protectedErrors(),
			responseText("404", "Requested student was not found."),
		),
	})
	doc.Paths["/api/admin/students/{id}/nudge"] = route("POST", Operation{
		Summary:    "Queue a manual nudge for a student",
		Tags:       []string{"Admin"},
		Security:   protected,
		Parameters: idParam("Student identifier."),
		Responses: mergeResponses(
			responseJSON("202", "Nudge accepted for delivery.", registry.refFor(manualNudgeResponse{})),
			protectedErrors(),
			responseText("400", "Student cannot receive a manual Telegram nudge."),
			responseText("404", "Requested student was not found."),
			responseText("502", "Downstream chat gateway failed to send."),
		),
	})
	doc.Paths["/api/admin/metrics"] = route("GET", Operation{
		Summary:  "Get dashboard metrics",
		Tags:     []string{"Admin"},
		Security: protected,
		Responses: mergeResponses(
			responseJSON("200", "Metrics summary.", registry.refFor(adminapi.MetricsSummary{})),
			protectedErrors(),
		),
	})
	doc.Paths["/api/admin/ai/usage"] = route("GET", Operation{
		Summary:  "Get AI usage summary",
		Tags:     []string{"Admin"},
		Security: protected,
		Responses: mergeResponses(
			responseJSON("200", "AI usage summary.", registry.refFor(adminapi.AIUsageSummary{})),
			protectedErrors(),
		),
	})
	doc.Paths["/api/admin/ai/budget-window"] = route("POST", Operation{
		Summary:     "Create or update the token budget window for the tenant",
		Tags:        []string{"Admin"},
		Security:    protected,
		RequestBody: jsonBody(registry.refFor(adminapi.UpsertTokenBudgetWindowRequest{})),
		Responses: mergeResponses(
			responseJSON("200", "Updated AI usage summary.", registry.refFor(adminapi.AIUsageSummary{})),
			protectedErrors(),
			responseText("400", "Request body is invalid."),
			responseText("404", "Token budget window could not be updated."),
		),
	})
	doc.Paths["/api/admin/parents/{id}"] = route("GET", Operation{
		Summary:    "Get parent summary",
		Tags:       []string{"Admin"},
		Security:   protected,
		Parameters: idParam("Parent identifier."),
		Responses: mergeResponses(
			responseJSON("200", "Parent summary payload.", registry.refFor(adminapi.ParentSummary{})),
			protectedErrors(),
			responseText("404", "Requested parent was not found."),
		),
	})

	doc.Components.Schemas = registry.schemas
	return doc, nil
}

type tenantRequiredErrorResponse struct {
	Error   string              `json:"error"`
	Tenants []auth.TenantOption `json:"tenants"`
}

func route(method string, operation Operation) *PathItem {
	item := &PathItem{}
	switch method {
	case "GET":
		item.Get = &operation
	case "POST":
		item.Post = &operation
	default:
		panic(fmt.Sprintf("unsupported method %q", method))
	}
	return item
}

func jsonBody(schema *Schema) *RequestBody {
	return &RequestBody{
		Required: true,
		Content: map[string]MediaType{
			"application/json": {Schema: schema},
		},
	}
}

func arrayOf(schema *Schema) *Schema {
	return &Schema{Type: "array", Items: schema}
}

func okJSON(description string, schema *Schema) Responses {
	return responseJSON("200", description, schema)
}

func responseJSON(status, description string, schema *Schema) Responses {
	return Responses{
		status: {
			Description: description,
			Content: map[string]MediaType{
				"application/json": {Schema: schema},
			},
		},
	}
}

func responseText(status, description string) Responses {
	return Responses{
		status: {
			Description: description,
			Content: map[string]MediaType{
				"text/plain": {Schema: &Schema{Type: "string"}},
			},
		},
	}
}

func responseMixed400(schema *Schema) Responses {
	return Responses{
		"400": {
			Description: "Request body is invalid, or tenant selection is required for this account.",
			Content: map[string]MediaType{
				"text/plain":       {Schema: &Schema{Type: "string"}},
				"application/json": {Schema: schema},
			},
		},
	}
}

func responseEmpty(status, description string) Responses {
	return Responses{
		status: {Description: description},
	}
}

func protectedErrors() Responses {
	return mergeResponses(
		responseText("401", "Request is not authenticated."),
		responseText("403", "Authenticated user is not allowed to access this resource."),
	)
}

func mergeResponses(groups ...Responses) Responses {
	out := Responses{}
	for _, group := range groups {
		for code, response := range group {
			out[code] = response
		}
	}
	return out
}
