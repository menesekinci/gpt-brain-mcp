package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/enes/project-brain-mcp/internal/app"
	"github.com/enes/project-brain-mcp/internal/audit"
	"github.com/enes/project-brain-mcp/internal/diagnostics"
	"github.com/enes/project-brain-mcp/internal/mcpserver"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		runDoctor(os.Args[2:])
		return
	}

	var (
		configPath = flag.String("config", "", "Path to project-brain.yml config file")
		addr       = flag.String("addr", "", "Listen address (overrides config)")
		issuerURL  = flag.String("issuer-url", "", "OAuth issuer URL, overrides config (e.g. https://xxx.trycloudflare.com)")
	)
	flag.Parse()

	cfg, err := app.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	listenAddr := cfg.Server.ListenAddr
	if *addr != "" {
		listenAddr = *addr
	}
	if *issuerURL != "" {
		cfg.Server.PublicBaseURL = *issuerURL
		cfg.Auth.IssuerURL = *issuerURL
		cfg.Auth.RedirectURI = *issuerURL + "/oauth/callback"
	}

	// Setup audit logger.
	home, _ := os.UserHomeDir()
	auditDir := os.Getenv("PROJECT_BRAIN_AUDIT_DIR")
	if auditDir == "" {
		auditDir = home + "/.project-brain/audit"
	}
	auditLogger, err := audit.NewLogger(cfg.Security.AuditLog, auditDir)
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	server, err := mcpserver.NewServer(cfg, auditLogger)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("  Project Brain MCP Server")
	fmt.Printf("  Version: 0.1.0\n")
	fmt.Printf("  Mode:    %s\n", cfg.Security.Mode)
	fmt.Printf("  Auth:    %s\n", cfg.Auth.Type)
	fmt.Println("========================================")

	if err := server.Serve(listenAddr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to project-brain.yml config file")
	publicURL := fs.String("public-url", "", "Public connector base URL, e.g. https://xxx.trycloudflare.com")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse doctor flags: %v", err)
	}

	cfg, err := app.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	report := diagnostics.Run(ctx, cfg, *configPath, *publicURL)
	fmt.Print(diagnostics.Format(report))
	if report.HasFailures() {
		os.Exit(1)
	}
}
