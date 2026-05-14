package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/enes/project-brain-mcp/internal/app"
	"github.com/enes/project-brain-mcp/internal/audit"
	"github.com/enes/project-brain-mcp/internal/auth"
	"github.com/enes/project-brain-mcp/internal/fsx"
	"github.com/enes/project-brain-mcp/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps an MCP server with project-brain dependencies.
type Server struct {
	mcpServer   *mcp.Server
	cfg         *app.Config
	roots       *fsx.Roots
	guard       *fsx.Guard
	auditLogger *audit.Logger
	authHandler http.Handler
	limiter     *security.RateLimiter
	oauthServer *auth.OAuthServer
}

// NewServer creates a new Project Brain MCP server.
func NewServer(cfg *app.Config, auditLogger *audit.Logger) (*Server, error) {
	roots := fsx.NewRoots(cfg.Roots)
	guard := fsx.NewGuard(cfg.Ignore)
	limiter := security.NewRateLimiter(100, 60) // 100 requests per minute

	authHandler := setupAuth(cfg)

	var oauthSrv *auth.OAuthServer
	if cfg.Auth.Type == "oauth" {
		issuer := cfg.Auth.IssuerURL
		if issuer == "" {
			issuer = cfg.Server.PublicBaseURL
		}
		redirectURI := cfg.Auth.RedirectURI
		if redirectURI == "" {
			redirectURI = issuer + "/oauth/callback"
		}
		tm := auth.NewTokenManager([]byte(cfg.Auth.JWTSecret), issuer, issuer+"/mcp/")
		oauthSrv = auth.NewOAuthServer(
			issuer,
			cfg.Auth.ClientID,
			redirectURI,
			cfg.Auth.OwnerSecret,
			tm,
			time.Duration(cfg.Auth.TokenExpiryMinutes)*time.Minute,
			5*time.Minute,
		)
	}

	opts := &mcp.ServerOptions{
		Instructions: "Project Brain MCP Server. Inspect local software projects and create markdown plans.",
	}
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "project-brain",
		Version: "0.1.0",
	}, opts)

	s := &Server{
		mcpServer:   mcpServer,
		cfg:         cfg,
		roots:       roots,
		guard:       guard,
		auditLogger: auditLogger,
		authHandler: authHandler,
		limiter:     limiter,
		oauthServer: oauthSrv,
	}

	s.registerTools()
	return s, nil
}

func setupAuth(cfg *app.Config) http.Handler {
	// For MVP, we chain noauth_dev + owner guard if configured.
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// placeholder
	})
	_ = base

	// Return a middleware-like handler; actual middleware applied in HTTP handler.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

// HTTPHandler returns the MCP streamable HTTP handler.
func (s *Server) HTTPHandler() http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{JSONResponse: true, Stateless: true, DisableLocalhostProtection: true})

	if s.cfg.Auth.Type == "oauth" && s.oauthServer != nil {
		// Wrap with bearer token middleware
		return s.oauthServer.RequireBearerTokenMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mcpHandler.ServeHTTP(w, r)
		}))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple auth check.
		if err := s.checkAuth(r); err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	})
}

func (s *Server) checkAuth(r *http.Request) error {
	if s.cfg.Auth.Type == "noauth_dev" {
		return nil
	}
	// TODO: implement OAuth / token validation.
	return nil
}

// jsonContent creates a JSON text content result and typed structured output.
func jsonContent[Out any](v Out) (*mcp.CallToolResult, Out, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		var zero Out
		return nil, zero, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, v, nil
}

// textContent creates a plain text result and typed structured output.
func textContent[Out any](text string, v Out) (*mcp.CallToolResult, Out, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, v, nil
}

// errorResult creates an error tool result.
func errorResult[Out any](msg string) (*mcp.CallToolResult, Out, error) {
	var zero Out
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, zero, nil
}

func (s *Server) checkToolRate(tool string) error {
	if s.limiter == nil {
		return nil
	}
	return s.limiter.Allow(tool)
}

func (s *Server) audit(tool, projectID, path, decision, reason string, bytesReturned int) {
	if s.auditLogger == nil {
		return
	}
	s.auditLogger.Log(audit.Event{
		Tool:          tool,
		ProjectID:     projectID,
		Path:          path,
		Decision:      decision,
		Reason:        reason,
		BytesReturned: bytesReturned,
	})
}

// Audit returns the audit logger for external use.
func (s *Server) Audit() *audit.Logger { return s.auditLogger }

// Serve runs the HTTP server (blocking).
func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()

	if s.cfg.Auth.Type == "oauth" && s.oauthServer != nil {
		s.oauthServer.RegisterOAuthHandlers(mux)
	}

	mux.Handle(s.cfg.Server.MCPPath+"/", s.HTTPHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	fmt.Printf("Project Brain MCP server listening on %s\n", addr)
	fmt.Printf("MCP endpoint: %s\n", s.cfg.Server.MCPPath)
	if s.cfg.Auth.Type == "oauth" {
		fmt.Printf("Auth mode: OAuth (issuer: %s)\n", s.oauthServer.IssuerURL)
	}
	return http.ListenAndServe(addr, mux)
}
