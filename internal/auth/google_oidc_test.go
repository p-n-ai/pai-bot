package auth

import "testing"

func TestRewriteLocalOIDCTransportEndpoints(t *testing.T) {
	doc := googleDiscoveryDocument{
		Issuer:                "http://localhost:4003",
		AuthorizationEndpoint: "http://localhost:4003/o/oauth2/v2/auth",
		TokenEndpoint:         "http://localhost:4003/oauth2/token",
		UserInfoEndpoint:      "http://localhost:4003/oauth2/v2/userinfo",
		JWKSURI:               "http://localhost:4003/oauth2/v3/certs",
	}

	rewritten := rewriteLocalOIDCTransportEndpoints(doc, "http://host.docker.internal:4003/.well-known/openid-configuration")

	if rewritten.Issuer != doc.Issuer {
		t.Fatalf("issuer = %q, want %q", rewritten.Issuer, doc.Issuer)
	}
	if rewritten.AuthorizationEndpoint != doc.AuthorizationEndpoint {
		t.Fatalf("authorization endpoint = %q, want %q", rewritten.AuthorizationEndpoint, doc.AuthorizationEndpoint)
	}
	if rewritten.TokenEndpoint != "http://host.docker.internal:4003/oauth2/token" {
		t.Fatalf("token endpoint = %q", rewritten.TokenEndpoint)
	}
	if rewritten.UserInfoEndpoint != "http://host.docker.internal:4003/oauth2/v2/userinfo" {
		t.Fatalf("userinfo endpoint = %q", rewritten.UserInfoEndpoint)
	}
	if rewritten.JWKSURI != "http://host.docker.internal:4003/oauth2/v3/certs" {
		t.Fatalf("jwks uri = %q", rewritten.JWKSURI)
	}
}

func TestRewriteLocalOIDCTransportEndpointsSkipsNonLocalIssuers(t *testing.T) {
	doc := googleDiscoveryDocument{
		Issuer:                "https://accounts.google.com",
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
		UserInfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
		JWKSURI:               "https://www.googleapis.com/oauth2/v3/certs",
	}

	rewritten := rewriteLocalOIDCTransportEndpoints(doc, "http://host.docker.internal:4003/.well-known/openid-configuration")

	if rewritten.Issuer != doc.Issuer ||
		rewritten.AuthorizationEndpoint != doc.AuthorizationEndpoint ||
		rewritten.TokenEndpoint != doc.TokenEndpoint ||
		rewritten.UserInfoEndpoint != doc.UserInfoEndpoint ||
		rewritten.JWKSURI != doc.JWKSURI {
		t.Fatalf("non-local discovery doc was rewritten: %#v", rewritten)
	}
}
