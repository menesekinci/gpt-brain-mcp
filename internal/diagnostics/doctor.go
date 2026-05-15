package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/enes/project-brain-mcp/internal/app"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	Name   string
	Status Status
	Detail string
	Fix    string
}

type Report struct {
	Checks []Check
}

func (r Report) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == StatusFail {
			return true
		}
	}
	return false
}

func Run(ctx context.Context, cfg *app.Config, configPath string, publicURL string) Report {
	var checks []Check
	checks = append(checks, checkConfig(configPath))
	checks = append(checks, checkRoots(cfg.Roots)...)
	checks = append(checks, checkCloudflared())

	local := checkLocalHealth(ctx, cfg.Server.ListenAddr)
	checks = append(checks, local)
	if local.Status != StatusPass {
		checks = append(checks, checkPort(cfg.Server.ListenAddr))
	}

	effectivePublicURL := strings.TrimSpace(publicURL)
	if effectivePublicURL == "" {
		effectivePublicURL = strings.TrimSpace(cfg.Auth.IssuerURL)
	}
	if effectivePublicURL == "" {
		effectivePublicURL = strings.TrimSpace(cfg.Server.PublicBaseURL)
	}
	checks = append(checks, checkOAuthIssuer(cfg, effectivePublicURL))
	checks = append(checks, checkPublicHealth(ctx, effectivePublicURL))

	return Report{Checks: checks}
}

func Format(report Report) string {
	var b strings.Builder
	b.WriteString("Project Brain Doctor\n\n")
	for _, check := range report.Checks {
		b.WriteString(statusSymbol(check.Status))
		b.WriteString(" ")
		b.WriteString(check.Name)
		if check.Detail != "" {
			b.WriteString("\n  ")
			b.WriteString(check.Detail)
		}
		if check.Fix != "" {
			b.WriteString("\n  Fix: ")
			b.WriteString(check.Fix)
		}
		b.WriteString("\n\n")
	}
	if report.HasFailures() {
		b.WriteString("Result: one or more checks failed.\n")
	} else {
		b.WriteString("Result: no blocking failures detected.\n")
	}
	return b.String()
}

func checkConfig(configPath string) Check {
	if strings.TrimSpace(configPath) == "" {
		return Check{Name: "Config", Status: StatusPass, Detail: "Using built-in default configuration."}
	}
	if _, err := os.Stat(configPath); err != nil {
		return Check{
			Name:   "Config",
			Status: StatusFail,
			Detail: fmt.Sprintf("Config file is not readable: %v", err),
			Fix:    "pass a valid --config path or copy configs/project-brain.example.yml to configs/project-brain.yml",
		}
	}
	return Check{Name: "Config", Status: StatusPass, Detail: "Config found: " + configPath}
}

func checkRoots(roots []app.RootConfig) []Check {
	if len(roots) == 0 {
		return []Check{{
			Name:   "Configured roots",
			Status: StatusFail,
			Detail: "No project roots are configured.",
			Fix:    "add at least one roots entry in the Project Brain config",
		}}
	}
	checks := make([]Check, 0, len(roots))
	for _, root := range roots {
		name := "Root access: " + root.ID
		info, err := os.Stat(root.Path)
		if err != nil {
			checks = append(checks, Check{
				Name:   name,
				Status: StatusFail,
				Detail: fmt.Sprintf("%s is not readable: %v", root.Path, err),
				Fix:    "choose an existing folder and make sure your Windows user can read it",
			})
			continue
		}
		if !info.IsDir() {
			checks = append(checks, Check{
				Name:   name,
				Status: StatusFail,
				Detail: root.Path + " is not a directory.",
				Fix:    "set roots[].path to a project parent directory",
			})
			continue
		}
		if _, err := os.ReadDir(root.Path); err != nil {
			checks = append(checks, Check{
				Name:   name,
				Status: StatusFail,
				Detail: fmt.Sprintf("%s exists but cannot be listed: %v", root.Path, err),
				Fix:    "grant read access to the configured root folder",
			})
			continue
		}
		mode := "writable planning root"
		if root.ReadOnly {
			mode = "read-only root"
		}
		checks = append(checks, Check{Name: name, Status: StatusPass, Detail: fmt.Sprintf("%s (%s)", root.Path, mode)})
	}
	return checks
}

func checkCloudflared() Check {
	if path, ok := findCloudflared(); ok {
		return Check{Name: "cloudflared", Status: StatusPass, Detail: "Found: " + path}
	}
	return Check{
		Name:   "cloudflared",
		Status: StatusWarn,
		Detail: "cloudflared was not found in PATH or %USERPROFILE%\\bin.",
		Fix:    "install cloudflared or use a named HTTPS endpoint that forwards to the local server",
	}
}

func findCloudflared() (string, bool) {
	for _, name := range []string{"cloudflared.exe", "cloudflared"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, "bin", "cloudflared.exe")
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func checkLocalHealth(ctx context.Context, listenAddr string) Check {
	healthURL := "http://" + normalizeLocalListenAddr(listenAddr) + "/healthz"
	status, body, err := get(ctx, healthURL)
	if err != nil {
		return Check{
			Name:   "Local MCP server",
			Status: StatusFail,
			Detail: fmt.Sprintf("Cannot reach %s: %v", healthURL, err),
			Fix:    "start Project Brain with server.exe --config configs\\project-brain.yml",
		}
	}
	if status != http.StatusOK {
		return Check{
			Name:   "Local MCP server",
			Status: StatusFail,
			Detail: fmt.Sprintf("%s returned HTTP %d: %s", healthURL, status, trim(body, 160)),
			Fix:    "restart the local server and check the terminal output",
		}
	}
	return Check{Name: "Local MCP server", Status: StatusPass, Detail: healthURL + " returned 200 OK."}
}

func checkPort(listenAddr string) Check {
	conn, err := net.DialTimeout("tcp", normalizeLocalListenAddr(listenAddr), 1500*time.Millisecond)
	if err != nil {
		return Check{
			Name:   "Listen port",
			Status: StatusWarn,
			Detail: normalizeLocalListenAddr(listenAddr) + " is not accepting TCP connections.",
			Fix:    "start the server or check whether the configured port is correct",
		}
	}
	_ = conn.Close()
	return Check{
		Name:   "Listen port",
		Status: StatusWarn,
		Detail: normalizeLocalListenAddr(listenAddr) + " accepts TCP connections, but /healthz did not pass.",
		Fix:    "make sure this port belongs to Project Brain, not another process",
	}
}

func checkOAuthIssuer(cfg *app.Config, publicURL string) Check {
	if cfg.Auth.Type != "oauth" {
		return Check{Name: "OAuth issuer", Status: StatusPass, Detail: "OAuth is not enabled."}
	}
	if strings.TrimSpace(publicURL) == "" {
		return Check{
			Name:   "OAuth issuer",
			Status: StatusFail,
			Detail: "OAuth is enabled but no public issuer URL is configured.",
			Fix:    "start with --issuer-url https://<your-public-host> or set auth.issuer_url",
		}
	}
	return Check{Name: "OAuth issuer", Status: StatusPass, Detail: "Using issuer: " + publicURL}
}

func checkPublicHealth(ctx context.Context, publicURL string) Check {
	if strings.TrimSpace(publicURL) == "" {
		return Check{
			Name:   "Public connector endpoint",
			Status: StatusWarn,
			Detail: "No public URL was provided, so tunnel/connector health was not checked.",
			Fix:    "run doctor with --public-url https://<your-public-host>",
		}
	}
	base, err := url.Parse(strings.TrimRight(publicURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return Check{
			Name:   "Public connector endpoint",
			Status: StatusFail,
			Detail: "Invalid public URL: " + publicURL,
			Fix:    "use a full HTTPS URL such as https://example.trycloudflare.com",
		}
	}
	healthURL := strings.TrimRight(publicURL, "/") + "/healthz"
	status, body, err := get(ctx, healthURL)
	if err != nil {
		return Check{
			Name:   "Public connector endpoint",
			Status: StatusFail,
			Detail: fmt.Sprintf("Cannot reach %s: %v", healthURL, err),
			Fix:    "restart cloudflared and reconnect the ChatGPT app if the tunnel URL changed",
		}
	}
	if status != http.StatusOK {
		return Check{
			Name:   "Public connector endpoint",
			Status: StatusFail,
			Detail: explainPublicHTTPFailure(status, body),
			Fix:    "restart cloudflared, confirm it forwards to http://127.0.0.1:3939, then refresh/reconnect the ChatGPT app",
		}
	}
	return Check{Name: "Public connector endpoint", Status: StatusPass, Detail: healthURL + " returned 200 OK."}
}

func get(ctx context.Context, rawURL string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, "", err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, string(buf[:n]), nil
}

func normalizeLocalListenAddr(listenAddr string) string {
	listenAddr = strings.TrimSpace(listenAddr)
	if listenAddr == "" {
		return "127.0.0.1:3939"
	}
	if strings.HasPrefix(listenAddr, "0.0.0.0:") {
		return "127.0.0.1:" + strings.TrimPrefix(listenAddr, "0.0.0.0:")
	}
	if strings.HasPrefix(listenAddr, ":") {
		return "127.0.0.1" + listenAddr
	}
	return listenAddr
}

func explainPublicHTTPFailure(status int, body string) string {
	body = trim(strings.ReplaceAll(body, "\n", " "), 240)
	switch status {
	case http.StatusBadGateway:
		return "HTTP 502 from the public endpoint. Cloudflare reached the tunnel edge, but cloudflared could not get a valid response from the local Project Brain service. Body: " + body
	case 530:
		if strings.Contains(body, "1033") {
			return "Cloudflare error 1033 from the public endpoint. The tunnel hostname exists, but no active tunnel connection is serving it. Body: " + body
		}
		return "HTTP 530 from Cloudflare. This usually means a tunnel/origin routing problem. Body: " + body
	default:
		return fmt.Sprintf("Public endpoint returned HTTP %d: %s", status, body)
	}
}

func statusSymbol(status Status) string {
	switch status {
	case StatusPass:
		return "[OK]"
	case StatusWarn:
		return "[WARN]"
	default:
		return "[FAIL]"
	}
}

func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
