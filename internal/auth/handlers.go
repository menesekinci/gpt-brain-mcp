package auth

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

//go:embed approve.html
var approveHTML string

// RegisterOAuthHandlers registers all OAuth-related HTTP handlers on the given mux.
func (s *OAuthServer) RegisterOAuthHandlers(mux *http.ServeMux) {
	// Protected Resource Metadata (RFC 9728)
	prm := &oauthex.ProtectedResourceMetadata{
		Resource:             s.IssuerURL + "/mcp/",
		AuthorizationServers: []string{s.IssuerURL},
		ScopesSupported:      DefaultScopes,
		BearerMethodsSupported: []string{"header"},
	}
	mux.Handle("/.well-known/oauth-protected-resource", auth.ProtectedResourceMetadataHandler(prm))

	// Authorization Server Metadata
	asm := &oauthex.AuthServerMeta{
		Issuer:                s.IssuerURL,
		AuthorizationEndpoint: s.IssuerURL + "/oauth/authorize",
		TokenEndpoint:         s.IssuerURL + "/oauth/token",
		JWKSURI:               s.IssuerURL + "/.well-known/jwks.json",
		ScopesSupported:       DefaultScopes,
		ResponseTypesSupported: []string{"code"},
		ResponseModesSupported: []string{"query"},
		GrantTypesSupported:    []string{"authorization_code"},
		CodeChallengeMethodsSupported: []string{"S256"},
	}
	mux.Handle("/.well-known/oauth-authorization-server", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(asm)
	}))

	// JWKS endpoint (symmetric key exposed for minimal setup)
	mux.Handle("/.well-known/jwks.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// For HMAC we return a minimal JWK. In production use asymmetric keys.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{},
		})
	}))

	// Authorization endpoint
	mux.Handle("/oauth/authorize", http.HandlerFunc(s.handleAuthorize))

	// Approval endpoint (POST from the approve form)
	mux.Handle("/oauth/approve", http.HandlerFunc(s.handleApprove))

	// Token endpoint
	mux.Handle("/oauth/token", http.HandlerFunc(s.handleToken))
}

func (s *OAuthServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeMethod := q.Get("code_challenge_method")

	if clientID != s.ClientID {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}
	if redirectURI == "" {
		http.Error(w, "redirect_uri required", http.StatusBadRequest)
		return
	}
	// Validate redirect URI: exact match to configured URI, or from allowed host
	if s.RedirectURI != "" && redirectURI != s.RedirectURI {
		if s.AllowedRedirectHost != "" {
			u, err := url.Parse(redirectURI)
			if err != nil || !strings.HasSuffix(u.Host, s.AllowedRedirectHost) {
				http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
			return
		}
	}
	if codeChallenge == "" {
		http.Error(w, "code_challenge required", http.StatusBadRequest)
		return
	}

	approveToken := s.GenerateApproveToken(state)

	// Render simple approve page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := strings.ReplaceAll(approveHTML, "{{ISSUER}}", s.IssuerURL)
	page = strings.ReplaceAll(page, "{{STATE}}", state)
	page = strings.ReplaceAll(page, "{{APPROVE_TOKEN}}", approveToken)
	page = strings.ReplaceAll(page, "{{REDIRECT_URI}}", redirectURI)
	page = strings.ReplaceAll(page, "{{CLIENT_ID}}", clientID)
	page = strings.ReplaceAll(page, "{{CODE_CHALLENGE}}", codeChallenge)
	page = strings.ReplaceAll(page, "{{CODE_METHOD}}", codeMethod)
	fmt.Fprint(w, page)
}

func (s *OAuthServer) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	state := r.FormValue("state")
	approveToken := r.FormValue("approve_token")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeChallenge := r.FormValue("code_challenge")
	codeMethod := r.FormValue("code_method")

	if !s.ValidateApproveToken(state, approveToken) {
		http.Error(w, "invalid approval token", http.StatusForbidden)
		return
	}

	code := s.GenerateCode(codeChallenge, codeMethod, redirectURI, clientID)
	redirectURL := BuildRedirectURL(redirectURI, code, state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *OAuthServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		writeTokenError(w, "unsupported_grant_type", "")
		return
	}

	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	accessToken, err := s.ExchangeCode(code, redirectURI, clientID, codeVerifier)
	if err != nil {
		writeTokenError(w, "invalid_grant", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(s.TokenExpiry.Seconds()),
		"scope":        strings.Join(DefaultScopes, " "),
	})
}

func writeTokenError(w http.ResponseWriter, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	body := map[string]string{"error": code}
	if desc != "" {
		body["error_description"] = desc
	}
	_ = json.NewEncoder(w).Encode(body)
}

// RequireBearerTokenMiddleware returns middleware for the MCP endpoint.
func (s *OAuthServer) RequireBearerTokenMiddleware() func(http.Handler) http.Handler {
	verifier := func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		claims, err := s.TokenManager.VerifyAccessToken(token)
		if err != nil {
			return nil, auth.ErrInvalidToken
		}
		return claims.ToAuthTokenInfo(), nil
	}
	return auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.IssuerURL + "/.well-known/oauth-protected-resource",
		Scopes:              DefaultScopes,
	})
}
