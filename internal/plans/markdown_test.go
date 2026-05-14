package plans

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	relPath, data, err := Generate("root:my-app", TypePlan, "Auth Refactor", "# Plan\n\nDetails here.")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.HasPrefix(relPath, ".chatgpt/plans/") {
		t.Errorf("expected path under .chatgpt/plans/, got %q", relPath)
	}
	if !strings.HasSuffix(relPath, "-auth-refactor.md") {
		t.Errorf("expected filename ending in -auth-refactor.md, got %q", relPath)
	}

	content := string(data)
	if !strings.Contains(content, "type: plan") {
		t.Errorf("expected frontmatter with type plan")
	}
	if !strings.Contains(content, "# Plan") {
		t.Errorf("expected body content")
	}
}

func TestGenerateAgentHandoff(t *testing.T) {
	relPath, data, err := GenerateAgentHandoff("root:my-app", "codex", "Implement Auth", "# Task\n\nDo this.")
	if err != nil {
		t.Fatalf("GenerateAgentHandoff failed: %v", err)
	}

	if !strings.HasPrefix(relPath, ".chatgpt/handoffs/") {
		t.Errorf("expected path under .chatgpt/handoffs/, got %q", relPath)
	}

	content := string(data)
	if !strings.Contains(content, "agent: codex") {
		t.Errorf("expected frontmatter with agent name")
	}
}

func TestGenerateKimiPrompt(t *testing.T) {
	content := KimiPromptTemplate(KimiPromptSpec{
		TaskTitle:          "Implement Auth",
		Objective:          "Implement the authentication change.",
		PlanPath:           ".chatgpt/plans/2026-05-14-auth.md",
		ContextFiles:       []string{"internal/auth/token.go"},
		Constraints:        []string{"Keep the public API stable."},
		AcceptanceCriteria: []string{"go test ./... passes."},
	})

	relPath, data, err := Generate("root:my-app", TypeKimi, "Implement Auth", content)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.HasPrefix(relPath, ".chatgpt/kimi/") {
		t.Errorf("expected path under .chatgpt/kimi/, got %q", relPath)
	}

	body := string(data)
	for _, want := range []string{
		"type: kimi_prompt",
		"Keep all planning notes, summaries, commit messages, and user-facing text in English.",
		"Project Brain MCP",
		".chatgpt/plans/2026-05-14-auth.md",
		"internal/auth/token.go",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected generated prompt to contain %q", want)
		}
	}
}

func TestGenerateImplementationPrompt(t *testing.T) {
	content := ImplementationPromptTemplate(ImplementationPromptSpec{
		TaskTitle:          "Implement Auth",
		Objective:          "Implement the authentication change.",
		PlanPath:           ".chatgpt/plans/2026-05-14-auth.md",
		ContextFiles:       []string{"internal/auth/token.go"},
		Constraints:        []string{"Keep the public API stable."},
		AcceptanceCriteria: []string{"go test ./... passes."},
	})

	relPath, data, err := Generate("root:my-app", TypeImplementationPrompt, "Implement Auth", content)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.HasPrefix(relPath, ".chatgpt/implementation-prompts/") {
		t.Errorf("expected path under .chatgpt/implementation-prompts/, got %q", relPath)
	}

	body := string(data)
	for _, want := range []string{
		"type: implementation_prompt",
		"You are the implementation agent for this repository.",
		"Project Brain MCP",
		".chatgpt/plans/2026-05-14-auth.md",
		"togpt.md",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected generated prompt to contain %q", want)
		}
	}
}

func TestGenerateQuickPlan(t *testing.T) {
	content := QuickPlanTemplate(QuickPlanSpec{
		TaskTitle:      "Fix Dashboard Empty State",
		Objective:      "Improve the dashboard empty state without changing data loading behavior.",
		CurrentContext: "The dashboard view already has a loading branch and an empty branch.",
		RelevantFiles:  []string{"app/dashboard/page.tsx"},
		Phases: []string{
			"P1.1 Inspect the current dashboard empty state.",
			"P1.2 Implement the scoped UI change.",
			"P1.3 Validate responsive rendering and tests.",
		},
		AcceptanceCriteria: []string{"The empty state renders clearly on mobile and desktop."},
		Tests:              []string{"Run the relevant frontend checks."},
		Risks:              []string{"Avoid changing data fetching semantics."},
	})

	relPath, data, err := Generate("root:my-app", TypeQuickPlan, "Fix Dashboard Empty State", content)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.HasPrefix(relPath, ".chatgpt/quick-plans/") {
		t.Errorf("expected path under .chatgpt/quick-plans/, got %q", relPath)
	}

	body := string(data)
	for _, want := range []string{
		"type: quick_plan",
		"## Current Context",
		"## Short Phased Plan",
		"## Acceptance Criteria",
		"app/dashboard/page.tsx",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected generated quick plan to contain %q", want)
		}
	}
}

func TestProjectBrainGuideAndAgentsTemplatesAreGeneric(t *testing.T) {
	combined := ProjectBrainGuideTemplate("both") + "\n" + ProjectAgentsTemplate()
	for _, banned := range []string{"Kimi", "GPT 5.5", "Code CLI"} {
		if strings.Contains(combined, banned) {
			t.Errorf("expected reusable templates not to contain %q", banned)
		}
	}
	for _, want := range []string{
		"Project Brain MCP",
		"planning assistant",
		"implementation agent",
		".chatgpt/",
		".ai/",
		"fromgpt.md",
		"togpt.md",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("expected reusable templates to contain %q", want)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Auth Refactor Plan", "auth-refactor-plan"},
		{"  Spaces  ", "spaces"},
		{"Special!@#Chars", "special-chars"},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
