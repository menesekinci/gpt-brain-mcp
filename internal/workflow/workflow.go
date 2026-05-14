package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	SchemaVersion = "1.0"
	GateManual    = "manual"

	StatusInProgress           = "in_progress"
	StatusReadyForFinalization = "ready_for_finalization"
	StatusCompleted            = "completed"

	PhasePending              = "pending"
	PhaseInProgress           = "in_progress"
	PhaseAwaitingUserApproval = "awaiting_user_approval"
	PhaseApproved             = "approved"
)

type PhaseDefinition struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	RequiredSections []string `json:"required_sections"`
	QualityChecklist []string `json:"quality_checklist"`
	Prompt           string   `json:"prompt"`
}

type PhaseState struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	ArtifactPath string `json:"artifact_path,omitempty"`
	CompletedAt  string `json:"completed_at,omitempty"`
	ApprovedAt   string `json:"approved_at,omitempty"`
}

type State struct {
	SchemaVersion      string       `json:"schema_version"`
	SessionID          string       `json:"session_id"`
	ProjectID          string       `json:"project_id"`
	Title              string       `json:"title"`
	OriginalUserIntent string       `json:"original_user_intent"`
	GateMode           string       `json:"gate_mode"`
	Status             string       `json:"status"`
	CurrentPhase       string       `json:"current_phase"`
	Phases             []PhaseState `json:"phases"`
	CreatedAt          string       `json:"created_at"`
	UpdatedAt          string       `json:"updated_at"`
}

type StartResult struct {
	State          State
	StatePath      string
	PhasePrompt    string
	ArtifactTarget string
}

type CompleteResult struct {
	State      State
	WrittenTo  string
	NextAction string
}

type ApproveResult struct {
	State      State
	NextPhase  *PhaseDefinition
	NextAction string
}

type FinalizeResult struct {
	State                 State
	MasterPlanPath        string
	ImplementationPrompts []string
}

var phaseDefinitions = []PhaseDefinition{
	{
		ID:    "01-intent",
		Title: "Intent",
		RequiredSections: []string{
			"Product Idea", "Target Users", "Core Problem", "Goals", "Non-Goals", "Constraints", "Success Criteria", "Assumptions", "Clarifying Questions",
		},
		QualityChecklist: []string{
			"Translate the user's raw idea into clear English.",
			"Separate confirmed facts from assumptions.",
			"End with only the open questions needed before research.",
		},
		Prompt: "Produce a focused intent artifact. Do not design the whole product yet.",
	},
	{
		ID:    "02-deep-search",
		Title: "Deep Search",
		RequiredSections: []string{
			"Research Questions", "Sources Checked", "Comparable Products", "Relevant Mechanics", "Technology Findings", "Market/Product Risks", "Unresolved Research Gaps",
		},
		QualityChecklist: []string{
			"Use the assistant's own research/web capability when available.",
			"Record source URLs or mark research gaps clearly.",
			"Do not add an MCP web-search requirement.",
		},
		Prompt: "Research the product space, related mechanics, and likely technology choices before making architecture decisions.",
	},
	{
		ID:    "03-tech-stack",
		Title: "Tech Stack",
		RequiredSections: []string{
			"Candidate Stacks", "Recommended Stack", "Rationale", "Packages And Runtime Choices", "Deployment Shape", "Integration Constraints", "Tradeoffs",
		},
		QualityChecklist: []string{
			"Compare viable options before recommending one.",
			"Explain why the recommended stack fits the mechanics and roadmap.",
			"Keep package choices concrete enough for implementation planning.",
		},
		Prompt: "Select and justify the stack after considering research and project constraints.",
	},
	{
		ID:    "04-mechanics",
		Title: "Mechanics",
		RequiredSections: []string{
			"User Roles", "Core Actions", "Feature Mechanics", "State Transitions", "Permissions", "Interaction Map", "Mechanics Dependencies",
		},
		QualityChecklist: []string{
			"Describe mechanics as user-visible behavior.",
			"Show how mechanics depend on each other.",
			"Capture permissions and state changes explicitly.",
		},
		Prompt: "Define the product mechanics deeply enough to drive database and architecture design.",
	},
	{
		ID:    "05-db-design",
		Title: "DB Design",
		RequiredSections: []string{
			"Entities", "Tables Or Collections", "Fields", "Relations", "Indexes", "Constraints", "Migrations", "Access/Security Model", "Example Queries",
		},
		QualityChecklist: []string{
			"Tie every entity back to a mechanic.",
			"Include relation cardinality and important indexes.",
			"Identify migration and access-control implications.",
		},
		Prompt: "Design the persistence model required by the approved mechanics.",
	},
	{
		ID:    "06-deep-dive",
		Title: "Deep Dive",
		RequiredSections: []string{
			"Architecture", "Modules", "API/Data Flow", "Background Jobs", "Integration Points", "Failure Modes", "Performance", "Security", "Observability",
		},
		QualityChecklist: []string{
			"Connect architecture to stack and data model decisions.",
			"Cover failure modes and security boundaries.",
			"Keep modules implementation-ready without writing code.",
		},
		Prompt: "Deeply define the architecture and operational behavior.",
	},
	{
		ID:    "07-theme-style-palette",
		Title: "Theme, Style, Color Palette",
		RequiredSections: []string{
			"Product Personality", "UI Principles", "Layout System", "Color Palette", "Typography", "Components", "Responsive Rules", "Accessibility Rules",
		},
		QualityChecklist: []string{
			"Use hex values for colors.",
			"Tie visual style to audience and product mechanics.",
			"Define responsive and accessibility expectations.",
		},
		Prompt: "Create a UI direction that can guide implementation without becoming a marketing page by default.",
	},
	{
		ID:    "08-roadmap",
		Title: "Road Map",
		RequiredSections: []string{
			"MVP", "P1 Roadmap", "P2 Roadmap", "P3 Roadmap", "Dependency Order", "Release Checkpoints", "Risk Burn-Down",
		},
		QualityChecklist: []string{
			"Order work by dependency and risk.",
			"Keep MVP smaller than the full product.",
			"Identify checkpoints that can be reviewed.",
		},
		Prompt: "Build a staged roadmap that can be converted into implementation phases.",
	},
	{
		ID:    "09-phases",
		Title: "Phases",
		RequiredSections: []string{
			"Implementation Breakdown", "P1.1", "P1.2", "P1.3", "Later Phases", "Acceptance Criteria", "Validation Plan",
		},
		QualityChecklist: []string{
			"Use P1.1, P1.2 style numbering.",
			"Each item includes objective, likely files/modules, acceptance criteria, and validation.",
			"Group implementation slices so each can become a prompt.",
		},
		Prompt: "Convert the roadmap into implementation-ready phases.",
	},
	{
		ID:    "10-review-test",
		Title: "Review & Test",
		RequiredSections: []string{
			"Review Standards", "Unit Tests", "Integration Tests", "End-To-End Tests", "Security Checks", "Accessibility Checks", "Performance Checks", "Test Data", "Regression Plan", "Final Acceptance Checklist",
		},
		QualityChecklist: []string{
			"Map tests to critical mechanics and risks.",
			"Include review criteria for implementation agents.",
			"Define final acceptance before implementation starts.",
		},
		Prompt: "Define review and validation standards for the implementation plan.",
	},
}

func Phases() []PhaseDefinition {
	out := make([]PhaseDefinition, len(phaseDefinitions))
	copy(out, phaseDefinitions)
	return out
}

func PhaseByID(id string) (PhaseDefinition, bool) {
	for _, phase := range phaseDefinitions {
		if phase.ID == id {
			return phase, true
		}
	}
	return PhaseDefinition{}, false
}

func Start(projectAbsPath, projectID, title, originalIntent string, now time.Time) (StartResult, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Planning Workflow"
	}
	originalIntent = strings.TrimSpace(originalIntent)
	if originalIntent == "" {
		return StartResult{}, fmt.Errorf("original_user_intent is required")
	}
	sessionID := now.UTC().Format("20060102-150405") + "-" + slugify(title)
	first := phaseDefinitions[0]
	ts := now.UTC().Format(time.RFC3339)
	state := State{
		SchemaVersion:      SchemaVersion,
		SessionID:          sessionID,
		ProjectID:          projectID,
		Title:              title,
		OriginalUserIntent: originalIntent,
		GateMode:           GateManual,
		Status:             StatusInProgress,
		CurrentPhase:       first.ID,
		CreatedAt:          ts,
		UpdatedAt:          ts,
	}
	for i, phase := range phaseDefinitions {
		status := PhasePending
		if i == 0 {
			status = PhaseInProgress
		}
		state.Phases = append(state.Phases, PhaseState{
			ID:     phase.ID,
			Title:  phase.Title,
			Status: status,
		})
	}
	if err := writeState(projectAbsPath, state); err != nil {
		return StartResult{}, err
	}
	return StartResult{
		State:          state,
		StatePath:      StateRelPath(sessionID),
		PhasePrompt:    PhasePrompt(state, first),
		ArtifactTarget: PhaseArtifactRelPath(sessionID, first.ID, first.Title),
	}, nil
}

func Load(projectAbsPath, sessionID string) (State, error) {
	if !validSessionID(sessionID) {
		return State{}, fmt.Errorf("invalid session_id")
	}
	data, err := os.ReadFile(filepath.Join(projectAbsPath, filepath.FromSlash(StateRelPath(sessionID))))
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func CurrentPhase(state State) (PhaseDefinition, error) {
	phase, ok := PhaseByID(state.CurrentPhase)
	if !ok {
		return PhaseDefinition{}, fmt.Errorf("unknown current phase: %s", state.CurrentPhase)
	}
	return phase, nil
}

func Complete(projectAbsPath, sessionID, phaseID, content string, now time.Time) (CompleteResult, error) {
	state, err := Load(projectAbsPath, sessionID)
	if err != nil {
		return CompleteResult{}, err
	}
	if state.Status != StatusInProgress {
		return CompleteResult{}, fmt.Errorf("workflow status must be %s, got %s", StatusInProgress, state.Status)
	}
	if phaseID != state.CurrentPhase {
		return CompleteResult{}, fmt.Errorf("phase %s is not current phase %s", phaseID, state.CurrentPhase)
	}
	idx := phaseIndex(state, phaseID)
	if idx < 0 {
		return CompleteResult{}, fmt.Errorf("phase not found: %s", phaseID)
	}
	if state.Phases[idx].Status != PhaseInProgress {
		return CompleteResult{}, fmt.Errorf("phase status must be %s, got %s", PhaseInProgress, state.Phases[idx].Status)
	}
	phase, _ := PhaseByID(phaseID)
	artifactPath := PhaseArtifactRelPath(sessionID, phase.ID, phase.Title)
	data := []byte(PhaseArtifactMarkdown(state, phase, content, now))
	if err := writeFile(projectAbsPath, artifactPath, data); err != nil {
		return CompleteResult{}, err
	}
	ts := now.UTC().Format(time.RFC3339)
	state.Phases[idx].Status = PhaseAwaitingUserApproval
	state.Phases[idx].ArtifactPath = artifactPath
	state.Phases[idx].CompletedAt = ts
	state.UpdatedAt = ts
	if err := writeState(projectAbsPath, state); err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{
		State:      state,
		WrittenTo:  artifactPath,
		NextAction: "Review this phase artifact with the user. Do not continue until the user explicitly approves or says continue.",
	}, nil
}

func Approve(projectAbsPath, sessionID, phaseID string, now time.Time) (ApproveResult, error) {
	state, err := Load(projectAbsPath, sessionID)
	if err != nil {
		return ApproveResult{}, err
	}
	if state.Status != StatusInProgress {
		return ApproveResult{}, fmt.Errorf("workflow status must be %s, got %s", StatusInProgress, state.Status)
	}
	if phaseID != state.CurrentPhase {
		return ApproveResult{}, fmt.Errorf("phase %s is not current phase %s", phaseID, state.CurrentPhase)
	}
	idx := phaseIndex(state, phaseID)
	if idx < 0 {
		return ApproveResult{}, fmt.Errorf("phase not found: %s", phaseID)
	}
	if state.Phases[idx].Status != PhaseAwaitingUserApproval {
		return ApproveResult{}, fmt.Errorf("phase status must be %s, got %s", PhaseAwaitingUserApproval, state.Phases[idx].Status)
	}
	ts := now.UTC().Format(time.RFC3339)
	state.Phases[idx].Status = PhaseApproved
	state.Phases[idx].ApprovedAt = ts
	state.UpdatedAt = ts
	if idx == len(state.Phases)-1 {
		state.Status = StatusReadyForFinalization
		state.CurrentPhase = ""
		if err := writeState(projectAbsPath, state); err != nil {
			return ApproveResult{}, err
		}
		return ApproveResult{
			State:      state,
			NextAction: "All phases are approved. Call finalize_planning_workflow to create the master dossier and implementation prompts.",
		}, nil
	}
	next := phaseDefinitions[idx+1]
	state.Phases[idx+1].Status = PhaseInProgress
	state.CurrentPhase = next.ID
	if err := writeState(projectAbsPath, state); err != nil {
		return ApproveResult{}, err
	}
	return ApproveResult{
		State:      state,
		NextPhase:  &next,
		NextAction: "The next phase is now open. Complete only this phase and wait for user approval again.",
	}, nil
}

func Finalize(projectAbsPath, sessionID, masterPlan string, implementationPrompts []ImplementationPrompt, now time.Time) (FinalizeResult, error) {
	state, err := Load(projectAbsPath, sessionID)
	if err != nil {
		return FinalizeResult{}, err
	}
	if state.Status != StatusReadyForFinalization {
		return FinalizeResult{}, fmt.Errorf("workflow must be %s before finalization, got %s", StatusReadyForFinalization, state.Status)
	}
	for _, phase := range state.Phases {
		if phase.Status != PhaseApproved {
			return FinalizeResult{}, fmt.Errorf("phase %s is not approved", phase.ID)
		}
	}
	masterPath := filepath.ToSlash(filepath.Join(".chatgpt", "workflows", sessionID, "final", "master-plan.md"))
	if err := writeFile(projectAbsPath, masterPath, []byte(FinalMarkdown(state, masterPlan, now))); err != nil {
		return FinalizeResult{}, err
	}
	var promptPaths []string
	for i, prompt := range implementationPrompts {
		title := strings.TrimSpace(prompt.Title)
		if title == "" {
			title = fmt.Sprintf("implementation-slice-%d", i+1)
		}
		rel := filepath.ToSlash(filepath.Join(".chatgpt", "implementation-prompts", now.UTC().Format("2006-01-02")+"-"+slugify(state.Title)+"-"+slugify(title)+".md"))
		if err := writeFile(projectAbsPath, rel, []byte(ImplementationPromptMarkdown(state, prompt, masterPath, now))); err != nil {
			return FinalizeResult{}, err
		}
		promptPaths = append(promptPaths, rel)
	}
	ts := now.UTC().Format(time.RFC3339)
	state.Status = StatusCompleted
	state.UpdatedAt = ts
	if err := writeState(projectAbsPath, state); err != nil {
		return FinalizeResult{}, err
	}
	return FinalizeResult{State: state, MasterPlanPath: masterPath, ImplementationPrompts: promptPaths}, nil
}

type ImplementationPrompt struct {
	Title              string   `json:"title"`
	Objective          string   `json:"objective"`
	ContextFiles       []string `json:"context_files,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

func PhasePrompt(state State, phase PhaseDefinition) string {
	return fmt.Sprintf(`# %s: %s

Create only the %s artifact for workflow session %s.

## Phase Purpose

%s

## Original User Intent

%s

## Required Sections

%s

## Quality Checklist

%s

## Required Closing Sections

- Summary
- Evidence
- Decisions
- Open Questions
- Risks
- Next-Phase Inputs

Manual gate: after completing this phase, wait for user approval before moving to the next phase.
`, phase.ID, phase.Title, phase.ID, state.SessionID, phase.Prompt, state.OriginalUserIntent, markdownList(phase.RequiredSections), markdownList(phase.QualityChecklist))
}

func PhaseArtifactMarkdown(state State, phase PhaseDefinition, content string, now time.Time) string {
	return fmt.Sprintf(`---
type: planning_workflow_phase
workflow_session_id: %s
phase_id: %s
project_id: %s
status: awaiting_user_approval
created_at: %s
generated_by: chatgpt
---

%s
`, state.SessionID, phase.ID, state.ProjectID, now.UTC().Format(time.RFC3339), strings.TrimSpace(content))
}

func FinalMarkdown(state State, content string, now time.Time) string {
	return fmt.Sprintf(`---
type: planning_workflow_final_dossier
workflow_session_id: %s
project_id: %s
status: completed
created_at: %s
generated_by: chatgpt
---

%s
`, state.SessionID, state.ProjectID, now.UTC().Format(time.RFC3339), strings.TrimSpace(content))
}

func ImplementationPromptMarkdown(state State, prompt ImplementationPrompt, masterPlanPath string, now time.Time) string {
	return fmt.Sprintf(`---
type: implementation_prompt
workflow_session_id: %s
project_id: %s
status: ready
created_at: %s
generated_by: chatgpt
---

# %s

You are the implementation agent for this repository.

## Source Dossier

%s

## Objective

%s

## Context Files

%s

## Constraints

%s

## Acceptance Criteria

%s

## Additional Notes

%s

## Required Final Response

- Files changed
- Tests or checks run
- Skipped validation and why
- Remaining risks or follow-up work
`, state.SessionID, state.ProjectID, now.UTC().Format(time.RFC3339), fallback(prompt.Title, "Implementation Slice"), masterPlanPath, fallback(prompt.Objective, "Implement the referenced slice from the master planning dossier."), markdownListOr(prompt.ContextFiles, "Inspect the repository and the source dossier before editing."), markdownListOr(prompt.Constraints, "Keep the change scoped to this implementation slice."), markdownListOr(prompt.AcceptanceCriteria, "Relevant checks pass and the source dossier requirements are satisfied."), fallback(prompt.Notes, "None."))
}

func StateRelPath(sessionID string) string {
	return filepath.ToSlash(filepath.Join(".chatgpt", "workflows", sessionID, "workflow.json"))
}

func PhaseArtifactRelPath(sessionID, phaseID, title string) string {
	return filepath.ToSlash(filepath.Join(".chatgpt", "workflows", sessionID, "phases", phaseID+"-"+slugify(title)+".md"))
}

func writeState(projectAbsPath string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(projectAbsPath, StateRelPath(state.SessionID), append(data, '\n'))
}

func writeFile(projectAbsPath, relPath string, data []byte) error {
	absPath := filepath.Join(projectAbsPath, filepath.FromSlash(relPath))
	if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(projectAbsPath)+string(filepath.Separator)) {
		return fmt.Errorf("path escapes project")
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, absPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func phaseIndex(state State, phaseID string) int {
	for i, phase := range state.Phases {
		if phase.ID == phaseID {
			return i
		}
	}
	return -1
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)
var sessionIDRe = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}-[a-z0-9][a-z0-9\-]*$`)

func validSessionID(sessionID string) bool {
	return sessionIDRe.MatchString(sessionID)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "workflow"
	}
	return s
}

func markdownList(items []string) string {
	return markdownListOr(items, "None.")
}

func markdownListOr(items []string, empty string) string {
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, "- "+item)
		}
	}
	if len(out) == 0 {
		return "- " + empty
	}
	return strings.Join(out, "\n")
}

func fallback(value, empty string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return empty
	}
	return value
}
