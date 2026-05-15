package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedTime() time.Time {
	return time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC)
}

func TestStartWorkflowCreatesStateAndFirstPhase(t *testing.T) {
	root := t.TempDir()
	result, err := Start(root, "personal-projects:app", "Social App", "Build a social app.", fixedTime())
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if result.State.SessionID != "20260514-080000-social-app" {
		t.Fatalf("unexpected session id: %s", result.State.SessionID)
	}
	if result.State.CurrentPhase != "01-intent" {
		t.Fatalf("expected first phase, got %s", result.State.CurrentPhase)
	}
	if result.State.Phases[0].Status != PhaseInProgress {
		t.Fatalf("expected first phase in progress, got %s", result.State.Phases[0].Status)
	}
	if !strings.HasPrefix(result.StatePath, ".chatgpt/workflows/") {
		t.Fatalf("state path outside workflows: %s", result.StatePath)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(result.StatePath))); err != nil {
		t.Fatalf("expected workflow.json to exist: %v", err)
	}
	if !strings.Contains(result.PhasePrompt, "Create only the 01-intent artifact") {
		t.Fatalf("expected first phase prompt, got %s", result.PhasePrompt)
	}
}

func TestCompletePhaseEnforcesCurrentPhaseAndOpensNextPhase(t *testing.T) {
	root := t.TempDir()
	start, err := Start(root, "personal-projects:app", "Social App", "Build a social app.", fixedTime())
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := Complete(root, start.State.SessionID, "02-deep-search", "# Wrong", fixedTime()); err == nil {
		t.Fatal("expected non-current phase completion to fail")
	}
	result, err := Complete(root, start.State.SessionID, "01-intent", "# Intent\n\n## Summary\n\nDone.", fixedTime())
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if result.State.Phases[0].Status != PhaseCompleted {
		t.Fatalf("expected completed phase, got %s", result.State.Phases[0].Status)
	}
	if result.State.CurrentPhase != "02-deep-search" {
		t.Fatalf("expected next phase to open, got %s", result.State.CurrentPhase)
	}
	if result.NextPhase == nil || result.NextPhase.ID != "02-deep-search" {
		t.Fatalf("expected next phase definition, got %+v", result.NextPhase)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(result.WrittenTo))); err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
	}
}

func TestFinalizeRequiresAllPhasesComplete(t *testing.T) {
	root := t.TempDir()
	start, _ := Start(root, "personal-projects:app", "Social App", "Build a social app.", fixedTime())
	if _, err := Finalize(root, start.State.SessionID, "# Master", nil, fixedTime()); err == nil {
		t.Fatal("expected finalize before all phases complete to fail")
	}
	for _, phase := range Phases() {
		if _, err := Complete(root, start.State.SessionID, phase.ID, "# "+phase.Title, fixedTime()); err != nil {
			t.Fatalf("Complete %s failed: %v", phase.ID, err)
		}
	}
	result, err := Finalize(root, start.State.SessionID, "# Master Plan", []ImplementationPrompt{{
		Title:              "P1.1 Auth",
		Objective:          "Implement P1.1.",
		AcceptanceCriteria: []string{"Tests pass."},
	}}, fixedTime())
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}
	if result.State.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", result.State.Status)
	}
	if result.MasterPlanPath != ".chatgpt/workflows/20260514-080000-social-app/final/master-plan.md" {
		t.Fatalf("unexpected master plan path: %s", result.MasterPlanPath)
	}
	if len(result.ImplementationPrompts) != 1 || !strings.HasPrefix(result.ImplementationPrompts[0], ".chatgpt/implementation-prompts/") {
		t.Fatalf("unexpected implementation prompt paths: %+v", result.ImplementationPrompts)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.ImplementationPrompts[0])))
	if err != nil {
		t.Fatalf("read implementation prompt: %v", err)
	}
	if !strings.Contains(string(data), "fromgpt.md") || !strings.Contains(string(data), "togpt.md") {
		t.Fatalf("expected implementation prompt to mention communication files")
	}
}

func TestPhaseTemplatesContainRequiredEnglishHeadings(t *testing.T) {
	for _, phase := range Phases() {
		for _, want := range []string{"Summary", "Evidence", "Decisions", "Open Questions", "Risks", "Next-Phase Inputs"} {
			if !strings.Contains(PhasePrompt(State{SessionID: "s", OriginalUserIntent: "intent"}, phase), want) {
				t.Fatalf("phase %s prompt missing %q", phase.ID, want)
			}
		}
		if len(phase.RequiredSections) == 0 {
			t.Fatalf("phase %s has no required sections", phase.ID)
		}
	}
}

func TestLoadRejectsInvalidSessionID(t *testing.T) {
	if _, err := Load(t.TempDir(), "../bad"); err == nil {
		t.Fatal("expected invalid session id to be rejected")
	}
}
