package apidocs

import (
	"encoding/json"
	"fmt"
)

type Document struct {
	OpenAPI    string     `json:"openapi"`
	Info       Info       `json:"info"`
	Servers    []Server   `json:"servers,omitempty"`
	Tags       []Tag      `json:"tags,omitempty"`
	Paths      Paths      `json:"paths"`
	Components Components `json:"components,omitempty"`
	Security   []Security `json:"security,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type Server struct {
	URL string `json:"url"`
}

type Tag struct {
	Name string `json:"name"`
}

type Paths map[string]*PathItem

type PathItem struct {
	Get  *Operation `json:"get,omitempty"`
	Post *Operation `json:"post,omitempty"`
}

type Operation struct {
	Summary     string       `json:"summary,omitempty"`
	Description string       `json:"description,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Parameters  []Parameter  `json:"parameters,omitempty"`
	RequestBody *RequestBody `json:"requestBody,omitempty"`
	Responses   Responses    `json:"responses"`
	Security    []Security   `json:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type Responses map[string]Response

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

type Security map[string][]string

type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`
}

func JSON() ([]byte, error) {
	doc, err := Build()
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal openapi document: %w", err)
	}
	return data, nil
}

func ScalarHTML(specURL string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>P&AI Bot API Docs</title>
	<style>
		html, body, #app {
			height: 100%%;
			margin: 0;
		}
	</style>
</head>
<body>
	<div id="app"></div>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
	<script>
		Scalar.createApiReference('#app', {
			url: %q,
		})
	</script>
</body>
</html>
`, specURL)
}
