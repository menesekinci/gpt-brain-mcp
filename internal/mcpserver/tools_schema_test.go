package mcpserver

import (
	"context"
	"testing"

	"github.com/enes/project-brain-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisteredToolsExposeOutputSchemas(t *testing.T) {
	cfg := app.DefaultConfig()
	cfg.Roots = []app.RootConfig{{
		ID:               "personal-projects",
		Name:             "Personal Projects",
		Path:             t.TempDir(),
		WritablePlanDirs: []string{".chatgpt", ".ai"},
	}}

	srv, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "schema-test-client"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect failed: %v", err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer clientSession.Close()

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("expected registered tools")
	}
	names := make(map[string]bool)
	for _, tool := range tools.Tools {
		names[tool.Name] = true
		if tool.OutputSchema == nil {
			t.Fatalf("tool %q has nil output schema", tool.Name)
		}
		if tool.Annotations == nil {
			t.Fatalf("tool %q has nil annotations", tool.Name)
		}
	}
	for _, name := range []string{
		"get_project_brain_guide",
		"list_roots",
		"list_projects",
		"inspect_project",
		"get_project_tree",
		"read_project_file",
		"search_project",
		"read_togpt_message",
	} {
		for _, tool := range tools.Tools {
			if tool.Name == name && !tool.Annotations.ReadOnlyHint {
				t.Fatalf("tool %q should be marked read-only", name)
			}
		}
	}
	for _, name := range []string{
		"bootstrap_project_agents_md",
		"start_planning_workflow",
		"complete_planning_phase",
		"approve_planning_phase",
		"finalize_planning_workflow",
		"append_fromgpt_message",
	} {
		for _, tool := range tools.Tools {
			if tool.Name == name {
				if tool.Annotations.ReadOnlyHint {
					t.Fatalf("tool %q should not be marked read-only", name)
				}
				if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
					t.Fatalf("tool %q should be marked non-destructive", name)
				}
			}
		}
	}
	for _, want := range []string{
		"get_project_brain_guide",
		"bootstrap_project_agents_md",
		"start_planning_workflow",
		"get_planning_workflow_status",
		"get_current_planning_phase",
		"complete_planning_phase",
		"approve_planning_phase",
		"finalize_planning_workflow",
		"read_togpt_message",
		"append_fromgpt_message",
	} {
		if !names[want] {
			t.Fatalf("expected tool %q to be registered", want)
		}
	}
	for _, removed := range []string{
		"create_project_analysis_note",
		"create_agent_plan",
		"create_agent_handoff",
		"create_implementation_prompt",
		"create_kimi_prompt",
	} {
		if names[removed] {
			t.Fatalf("legacy freeform planning tool %q should not be registered", removed)
		}
	}
}
