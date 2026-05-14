package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enes/project-brain-mcp/internal/app"
)

func newTestServer(t *testing.T, rootPath string) *Server {
	t.Helper()
	cfg := app.DefaultConfig()
	cfg.Roots = []app.RootConfig{{
		ID:               "personal-projects",
		Name:             "Personal Projects",
		Path:             rootPath,
		WritablePlanDirs: []string{".chatgpt", ".ai"},
		ReadOnly:         false,
	}}
	srv, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	return srv
}

func TestGetProjectBrainGuide(t *testing.T) {
	srv := newTestServer(t, t.TempDir())

	_, out, err := srv.handleGetProjectBrainGuide(context.Background(), nil, GetProjectBrainGuideInput{Audience: "both"})
	if err != nil {
		t.Fatalf("handleGetProjectBrainGuide failed: %v", err)
	}
	if out.Title == "" || out.Guide == "" || out.RecommendedUsage == "" {
		t.Fatalf("expected populated guide output, got %+v", out)
	}
	for _, want := range []string{"Project Brain MCP", "planning assistant", "implementation agent"} {
		if !strings.Contains(out.Guide, want) {
			t.Errorf("expected guide to contain %q", want)
		}
	}
}

func TestBootstrapProjectAgentsMD(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "app")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	srv := newTestServer(t, root)

	_, out, err := srv.handleBootstrapProjectAgentsMD(context.Background(), nil, BootstrapProjectAgentsInput{
		ProjectID: "personal-projects:app",
	})
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if out.WrittenTo != "AGENTS.md" || out.Status != "created" {
		t.Fatalf("unexpected bootstrap output: %+v", out)
	}
	data, err := os.ReadFile(filepath.Join(projectPath, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(data), "Project Brain MCP") {
		t.Fatalf("expected AGENTS.md to mention Project Brain MCP")
	}

	if err := os.WriteFile(filepath.Join(projectPath, "AGENTS.md"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write existing AGENTS.md: %v", err)
	}
	_, out, err = srv.handleBootstrapProjectAgentsMD(context.Background(), nil, BootstrapProjectAgentsInput{
		ProjectID: "personal-projects:app",
	})
	if err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}
	if out.Status != "skipped_existing" {
		t.Fatalf("expected skipped_existing, got %+v", out)
	}
	data, _ = os.ReadFile(filepath.Join(projectPath, "AGENTS.md"))
	if string(data) != "existing\n" {
		t.Fatalf("expected existing AGENTS.md to be preserved, got %q", string(data))
	}

	_, out, err = srv.handleBootstrapProjectAgentsMD(context.Background(), nil, BootstrapProjectAgentsInput{
		ProjectID: "personal-projects:app",
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("overwrite bootstrap failed: %v", err)
	}
	if out.Status != "overwritten" {
		t.Fatalf("expected overwritten, got %+v", out)
	}
	data, _ = os.ReadFile(filepath.Join(projectPath, "AGENTS.md"))
	if !strings.Contains(string(data), "Project Brain MCP") {
		t.Fatalf("expected overwritten AGENTS.md to contain template")
	}
}

func TestCreateImplementationPrompt(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "app")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	srv := newTestServer(t, root)

	_, out, err := srv.handleCreateImplementationPrompt(context.Background(), nil, CreateImplementationPromptInput{
		ProjectID:          "personal-projects:app",
		TaskTitle:          "Implement Auth",
		Objective:          "Implement the authentication change.",
		PlanPath:           ".chatgpt/plans/2026-05-14-auth.md",
		ContextFiles:       []string{"internal/auth/token.go"},
		AcceptanceCriteria: []string{"Relevant tests pass."},
	})
	if err != nil {
		t.Fatalf("create implementation prompt failed: %v", err)
	}
	if !strings.HasPrefix(out.WrittenTo, ".chatgpt/implementation-prompts/") {
		t.Fatalf("expected implementation prompt path, got %q", out.WrittenTo)
	}
	data, err := os.ReadFile(filepath.Join(projectPath, filepath.FromSlash(out.WrittenTo)))
	if err != nil {
		t.Fatalf("read implementation prompt: %v", err)
	}
	if !strings.Contains(string(data), "You are the implementation agent for this repository.") {
		t.Fatalf("expected generic implementation prompt content")
	}
}
