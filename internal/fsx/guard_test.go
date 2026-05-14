package fsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/enes/project-brain-mcp/internal/app"
)

func TestGuardIsPathAllowed(t *testing.T) {
	guard := NewGuard(app.IgnoreConfig{
		Dirs:  []string{"node_modules", ".git"},
		Files: []string{".env", "*.pem"},
	})
	root := app.RootConfig{Path: "/projects", WritablePlanDirs: []string{".chatgpt"}}

	tests := []struct {
		path    string
		allowed bool
	}{
		{"README.md", true},
		{"src/main.go", true},
		{".chatgpt/plan.md", true},
		{"../.ssh/id_rsa", false},
		{".env", false},
		{"secrets.env", false},
		{"config.pem", false},
		{"node_modules/foo", false},
		{".git/config", false},
	}

	for _, tt := range tests {
		err := guard.IsPathAllowed(tt.path, "read", root)
		if tt.allowed && err != nil {
			t.Errorf("expected %s to be allowed, got: %v", tt.path, err)
		}
		if !tt.allowed && err == nil {
			t.Errorf("expected %s to be blocked", tt.path)
		}
	}
}

func TestGuardIsWriteAllowed(t *testing.T) {
	guard := NewGuard(app.IgnoreConfig{})
	root := app.RootConfig{Path: "/projects", WritablePlanDirs: []string{".chatgpt", ".ai"}, ReadOnly: false}

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "my-app")
	os.MkdirAll(projectPath, 0o755)

	tests := []struct {
		absPath string
		allowed bool
	}{
		{filepath.Join(projectPath, "AGENTS.md"), true},
		{filepath.Join(projectPath, ".chatgpt/plan.md"), true},
		{filepath.Join(projectPath, ".ai/note.md"), true},
		{filepath.Join(projectPath, "src/main.go"), false},
		{filepath.Join(projectPath, "src/AGENTS.md"), false},
		{filepath.Join(projectPath, "agents.md"), false},
		{filepath.Join(projectPath, "../other.md"), false},
		{filepath.Join(tmpDir, "other-project/.chatgpt/plan.md"), false},
	}

	for _, tt := range tests {
		err := guard.IsWriteAllowed(tt.absPath, projectPath, root)
		if tt.allowed && err != nil {
			t.Errorf("expected %s to be allowed, got: %v", tt.absPath, err)
		}
		if !tt.allowed && err == nil {
			t.Errorf("expected %s to be blocked", tt.absPath)
		}
	}
}

func TestGuardIsSensitivePath(t *testing.T) {
	guard := NewGuard(app.IgnoreConfig{})

	tests := []struct {
		path      string
		sensitive bool
	}{
		{"/home/user/.ssh/id_rsa", true},
		{"/home/user/.aws/credentials", true},
		{"/projects/app/.env", true},
		{"/projects/app/config.json", false},
		{"/projects/app/secrets.yaml", true},
	}

	for _, tt := range tests {
		got := guard.IsSensitivePath(tt.path)
		if got != tt.sensitive {
			t.Errorf("IsSensitivePath(%q) = %v, want %v", tt.path, got, tt.sensitive)
		}
	}
}
