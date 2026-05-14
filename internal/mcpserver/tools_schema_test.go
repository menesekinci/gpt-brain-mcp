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
	}
	for _, want := range []string{
		"get_project_brain_guide",
		"bootstrap_project_agents_md",
		"create_implementation_prompt",
		"create_kimi_prompt",
	} {
		if !names[want] {
			t.Fatalf("expected tool %q to be registered", want)
		}
	}
}
