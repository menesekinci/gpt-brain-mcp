package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/enes/project-brain-mcp/internal/fsx"
	"github.com/enes/project-brain-mcp/internal/plans"
	"github.com/enes/project-brain-mcp/internal/project"
	"github.com/enes/project-brain-mcp/internal/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ===== Tool Input Types =====

type ListRootsInput struct{}

type ListProjectsInput struct {
	RootID   string `json:"root_id" jsonschema:"ID of the root to list projects from"`
	MaxDepth int    `json:"max_depth,omitempty" jsonschema:"Maximum depth to search (default 2)"`
}

type InspectProjectInput struct {
	ProjectID        string `json:"project_id" jsonschema:"Project identifier (root_id:project_name)"`
	IncludeTree      *bool  `json:"include_tree,omitempty" jsonschema:"Include file tree preview"`
	IncludeManifests *bool  `json:"include_manifests,omitempty" jsonschema:"Include manifest summaries"`
	IncludeGit       *bool  `json:"include_git,omitempty" jsonschema:"Include git state"`
}

type GetProjectTreeInput struct {
	ProjectID  string `json:"project_id" jsonschema:"Project identifier"`
	MaxEntries int    `json:"max_entries,omitempty" jsonschema:"Maximum entries to return (default 500)"`
	Depth      int    `json:"depth,omitempty" jsonschema:"Maximum tree depth (default 4)"`
}

type ReadProjectFileInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	Path      string `json:"path" jsonschema:"Relative file path within the project"`
}

type SearchProjectInput struct {
	ProjectID  string `json:"project_id" jsonschema:"Project identifier"`
	Query      string `json:"query" jsonschema:"Search query string"`
	Glob       string `json:"glob,omitempty" jsonschema:"Optional glob filter (e.g., **/*.go)"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results (default 50)"`
}

type CreateAnalysisNoteInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	Title     string `json:"title" jsonschema:"Title for the analysis note"`
	Content   string `json:"content" jsonschema:"Markdown content of the analysis"`
}

type CreateAgentPlanInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	PlanTitle string `json:"plan_title" jsonschema:"Title for the plan"`
	Goal      string `json:"goal" jsonschema:"Goal or objective of the plan"`
	Content   string `json:"content,omitempty" jsonschema:"Optional full markdown content; if empty a template is used"`
}

type CreateAgentHandoffInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	AgentName string `json:"agent_name" jsonschema:"Target agent name (e.g., codex)"`
	TaskTitle string `json:"task_title" jsonschema:"Task title"`
	Content   string `json:"content" jsonschema:"Markdown content for the handoff"`
}

type GetProjectBrainGuideInput struct {
	Audience string `json:"audience,omitempty" jsonschema:"Optional audience: planner, implementation_agent, or both. Defaults to both."`
}

type BootstrapProjectAgentsInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema:"Whether to overwrite an existing project-root AGENTS.md file"`
}

type CreateImplementationPromptInput struct {
	ProjectID          string   `json:"project_id" jsonschema:"Project identifier"`
	TaskTitle          string   `json:"task_title" jsonschema:"Short task title for the implementation prompt"`
	Objective          string   `json:"objective" jsonschema:"Implementation objective for the downstream implementation agent"`
	PlanPath           string   `json:"plan_path,omitempty" jsonschema:"Optional relative path to the ChatGPT-authored source plan"`
	ContextFiles       []string `json:"context_files,omitempty" jsonschema:"Optional relative files the implementation agent should inspect first"`
	Constraints        []string `json:"constraints,omitempty" jsonschema:"Optional implementation constraints"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty" jsonschema:"Optional acceptance criteria"`
	Notes              string   `json:"notes,omitempty" jsonschema:"Optional extra notes for the implementation agent"`
}

type CreateKimiPromptInput = CreateImplementationPromptInput

type CreateQuickPlanInput struct {
	ProjectID                  string   `json:"project_id" jsonschema:"Project identifier"`
	TaskTitle                  string   `json:"task_title" jsonschema:"Short task title for the quick plan"`
	Objective                  string   `json:"objective" jsonschema:"Scoped implementation objective. Use full planning workflow instead for product/architecture/migration work."`
	CurrentContext             string   `json:"current_context" jsonschema:"Summary of inspected project context and existing patterns"`
	RelevantFiles              []string `json:"relevant_files,omitempty" jsonschema:"Relevant files or directories discovered during inspection"`
	Phases                     []string `json:"phases,omitempty" jsonschema:"Three to five short implementation phases"`
	AcceptanceCriteria         []string `json:"acceptance_criteria,omitempty" jsonschema:"Acceptance criteria for the scoped change"`
	Tests                      []string `json:"tests,omitempty" jsonschema:"Tests or checks to run"`
	Risks                      []string `json:"risks,omitempty" jsonschema:"Risks or edge cases"`
	Notes                      string   `json:"notes,omitempty" jsonschema:"Optional extra notes"`
	CreateImplementationPrompt bool     `json:"create_implementation_prompt,omitempty" jsonschema:"Whether to also create an implementation prompt for this quick plan"`
}

type StartPlanningWorkflowInput struct {
	ProjectID          string `json:"project_id" jsonschema:"Project identifier"`
	Title              string `json:"title" jsonschema:"Short title for this planning workflow"`
	OriginalUserIntent string `json:"original_user_intent" jsonschema:"Raw user idea or product intent. It may be in any language; artifacts must be English."`
}

type PlanningWorkflowSessionInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	SessionID string `json:"session_id" jsonschema:"Planning workflow session identifier"`
}

type CompletePlanningPhaseInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	SessionID string `json:"session_id" jsonschema:"Planning workflow session identifier"`
	PhaseID   string `json:"phase_id" jsonschema:"Current phase identifier"`
	Content   string `json:"content" jsonschema:"English markdown content for the current phase artifact"`
}

type FinalizePlanningWorkflowInput struct {
	ProjectID             string                          `json:"project_id" jsonschema:"Project identifier"`
	SessionID             string                          `json:"session_id" jsonschema:"Planning workflow session identifier"`
	MasterPlan            string                          `json:"master_plan" jsonschema:"English consolidated master planning dossier markdown"`
	ImplementationPrompts []workflow.ImplementationPrompt `json:"implementation_prompts" jsonschema:"Implementation prompt slices to write after the workflow is complete"`
}

type AppendFromGPTMessageInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	Title     string `json:"title,omitempty" jsonschema:"Optional short title for the message"`
	Message   string `json:"message" jsonschema:"English markdown message, revision, or follow-up from the planning assistant"`
}

type ReadToGPTMessageInput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
}

type RootInfo struct {
	ID       string `json:"id" jsonschema:"Configured root identifier"`
	Name     string `json:"name" jsonschema:"Human readable root name"`
	PathHint string `json:"path_hint" jsonschema:"Configured path for this root"`
	Mode     string `json:"mode" jsonschema:"Server security mode applied to this root"`
}

type ListRootsOutput struct {
	Roots []RootInfo `json:"roots" jsonschema:"Configured project roots available to the MCP client"`
}

type ProjectInfo struct {
	ProjectID     string   `json:"project_id" jsonschema:"Project identifier to use in later tool calls"`
	Name          string   `json:"name" jsonschema:"Project directory name"`
	RelativePath  string   `json:"relative_path" jsonschema:"Path relative to the configured root"`
	DetectedStack []string `json:"detected_stack" jsonschema:"Detected languages and frameworks"`
	HasGit        bool     `json:"has_git" jsonschema:"Whether a .git directory was detected"`
}

type ListProjectsOutput struct {
	Projects []ProjectInfo `json:"projects" jsonschema:"Projects discovered under the selected root"`
}

type InspectProjectOutput struct {
	Inspection *project.InspectResult `json:"inspection" jsonschema:"Structured read-only project inspection result"`
}

type GetProjectTreeOutput struct {
	Entries []project.TreeEntry `json:"entries" jsonschema:"Filtered file tree entries"`
	Total   int                 `json:"total" jsonschema:"Number of returned tree entries"`
}

type ReadProjectFileOutput struct {
	ProjectID string `json:"project_id" jsonschema:"Project identifier"`
	Path      string `json:"path" jsonschema:"Relative file path read from the project"`
	Content   string `json:"content" jsonschema:"Redacted text file content"`
}

type SearchProjectOutput struct {
	Results []fsx.SearchResult `json:"results" jsonschema:"Matching file lines"`
	Total   int                `json:"total" jsonschema:"Number of returned matches"`
}

type WriteArtifactOutput struct {
	WrittenTo string `json:"written_to" jsonschema:"Relative path of the markdown artifact that was written"`
	Status    string `json:"status" jsonschema:"Write status"`
}

type ProjectBrainGuideOutput struct {
	Title            string `json:"title" jsonschema:"Guide title"`
	Version          string `json:"version" jsonschema:"Guide version"`
	Guide            string `json:"guide" jsonschema:"Reusable Project Brain MCP operating guide"`
	RecommendedUsage string `json:"recommended_usage" jsonschema:"How to use this guide in a conversation"`
}

type BootstrapProjectAgentsOutput struct {
	WrittenTo string `json:"written_to" jsonschema:"Relative path of the project-root AGENTS.md file"`
	Status    string `json:"status" jsonschema:"created, skipped_existing, or overwritten"`
	NextSteps string `json:"next_steps" jsonschema:"Recommended next steps after bootstrapping agent instructions"`
}

type ImplementationPromptOutput struct {
	WrittenTo       string `json:"written_to" jsonschema:"Relative path of the implementation prompt that was written"`
	Status          string `json:"status" jsonschema:"Write status"`
	CommandGuidance string `json:"command_guidance" jsonschema:"Generic guidance for passing the prompt to a downstream implementation agent"`
}

type KimiPromptOutput = ImplementationPromptOutput

type QuickPlanOutput struct {
	PlanPath                 string `json:"plan_path" jsonschema:"Relative path of the quick plan markdown file"`
	Status                   string `json:"status" jsonschema:"Write status"`
	ImplementationPromptPath string `json:"implementation_prompt_path,omitempty" jsonschema:"Relative path of the optional implementation prompt"`
	ModeGuidance             string `json:"mode_guidance" jsonschema:"Guidance for when to use full workflow instead"`
}

type StartPlanningWorkflowOutput struct {
	SessionID      string         `json:"session_id" jsonschema:"Planning workflow session identifier"`
	StatePath      string         `json:"state_path" jsonschema:"Relative path to workflow.json"`
	CurrentPhase   string         `json:"current_phase" jsonschema:"Current open phase identifier"`
	PhasePrompt    string         `json:"phase_prompt" jsonschema:"Prompt/contract for the current phase only"`
	ArtifactTarget string         `json:"artifact_target" jsonschema:"Relative target path for the current phase artifact"`
	State          workflow.State `json:"state" jsonschema:"Workflow state snapshot"`
}

type PlanningWorkflowStatusOutput struct {
	State workflow.State `json:"state" jsonschema:"Workflow state snapshot"`
}

type CurrentPlanningPhaseOutput struct {
	SessionID      string                   `json:"session_id" jsonschema:"Planning workflow session identifier"`
	CurrentPhase   workflow.PhaseDefinition `json:"current_phase" jsonschema:"Current phase definition"`
	PhasePrompt    string                   `json:"phase_prompt" jsonschema:"Prompt/contract for the current phase only"`
	ArtifactTarget string                   `json:"artifact_target" jsonschema:"Relative target path for the current phase artifact"`
	State          workflow.State           `json:"state" jsonschema:"Workflow state snapshot"`
}

type CompletePlanningPhaseOutput struct {
	SessionID   string                    `json:"session_id" jsonschema:"Planning workflow session identifier"`
	PhaseID     string                    `json:"phase_id" jsonschema:"Completed phase identifier"`
	WrittenTo   string                    `json:"written_to" jsonschema:"Relative path of the phase artifact that was written"`
	Status      string                    `json:"status" jsonschema:"Workflow status after completion"`
	NextPhase   *workflow.PhaseDefinition `json:"next_phase,omitempty" jsonschema:"Next phase definition if another phase is open"`
	PhasePrompt string                    `json:"phase_prompt,omitempty" jsonschema:"Prompt/contract for the next phase only"`
	NextAction  string                    `json:"next_action" jsonschema:"Next workflow action for the assistant"`
	State       workflow.State            `json:"state" jsonschema:"Workflow state snapshot"`
}

type FinalizePlanningWorkflowOutput struct {
	SessionID             string         `json:"session_id" jsonschema:"Planning workflow session identifier"`
	MasterPlanPath        string         `json:"master_plan_path" jsonschema:"Relative path of the final master planning dossier"`
	ImplementationPrompts []string       `json:"implementation_prompts" jsonschema:"Relative paths of generated implementation prompts"`
	Status                string         `json:"status" jsonschema:"Workflow status after finalization"`
	State                 workflow.State `json:"state" jsonschema:"Workflow state snapshot"`
}

type ReadToGPTMessageOutput struct {
	Path    string `json:"path" jsonschema:"Relative path read from the project"`
	Status  string `json:"status" jsonschema:"found or missing"`
	Content string `json:"content" jsonschema:"Redacted togpt.md content"`
}

type AppendFromGPTMessageOutput struct {
	WrittenTo string `json:"written_to" jsonschema:"Relative path of fromgpt.md"`
	Status    string `json:"status" jsonschema:"created or appended"`
	Timestamp string `json:"timestamp" jsonschema:"UTC timestamp added to the message"`
}

// ===== Tool Registration =====

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_project_brain_guide",
		Description: "Returns a neutral English context summary for Project Brain MCP. Read-only. Does not read project files, write files, or execute commands.",
		Annotations: readOnlyTool("Project Brain guide"),
	}, s.handleGetProjectBrainGuide)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_roots",
		Description: "Lists configured project roots that this MCP server is allowed to access.",
		Annotations: readOnlyTool("List roots"),
	}, s.handleListRoots)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_projects",
		Description: "Lists software projects under a configured root. Does not read file contents.",
		Annotations: readOnlyTool("List projects"),
	}, s.handleListProjects)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "inspect_project",
		Description: "Returns a structured read-only overview of a selected project, including detected stack, important files, manifest summaries, and warnings.",
		Annotations: readOnlyTool("Inspect project"),
	}, s.handleInspectProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_project_tree",
		Description: "Returns a filtered file tree for a project, respecting ignore rules and depth limits.",
		Annotations: readOnlyTool("Get project tree"),
	}, s.handleGetProjectTree)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "read_project_file",
		Description: "Reads the contents of a single allowed file from a project. Binary files, secrets, and oversized files are rejected.",
		Annotations: readOnlyTool("Read project file"),
	}, s.handleReadProjectFile)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_project",
		Description: "Searches for a query string across text files in a project. Returns matching lines with file paths and line numbers.",
		Annotations: readOnlyTool("Search project"),
	}, s.handleSearchProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "bootstrap_project_agents_md",
		Description: "Use this once per project to create a standard English project-root AGENTS.md explaining the Project Brain MCP workflow to downstream implementation agents. This tool writes only AGENTS.md and does not modify source code.",
		Annotations: planningWriteTool("Bootstrap AGENTS.md"),
	}, s.handleBootstrapProjectAgentsMD)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_quick_plan",
		Description: "Creates one short phased implementation plan for small or medium scoped work after project inspection. Do not use for product planning, architecture, migrations, auth/permissions, billing, or broad multi-module features; use start_planning_workflow instead.",
		Annotations: planningWriteTool("Create quick plan"),
	}, s.handleCreateQuickPlan)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "start_planning_workflow",
		Description: "Starts a strict automatic multi-phase planning workflow under .chatgpt/workflows. Use this for serious product/project planning instead of trying to produce the whole plan in one answer.",
		Annotations: planningWriteTool("Start planning workflow"),
	}, s.handleStartPlanningWorkflow)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_planning_workflow_status",
		Description: "Returns the current state of a planning workflow session. Read-only.",
		Annotations: readOnlyTool("Get planning workflow status"),
	}, s.handleGetPlanningWorkflowStatus)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_current_planning_phase",
		Description: "Returns the active phase contract, required sections, quality checklist, prompt, and target artifact path. Read-only.",
		Annotations: readOnlyTool("Get current planning phase"),
	}, s.handleGetCurrentPlanningPhase)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "complete_planning_phase",
		Description: "Writes the current phase artifact, marks it complete, and opens exactly the next phase. It keeps phases separate without requiring a separate approval tool.",
		Annotations: planningWriteTool("Complete planning phase"),
	}, s.handleCompletePlanningPhase)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "finalize_planning_workflow",
		Description: "Writes the final master planning dossier and implementation prompts after all planning phases are complete.",
		Annotations: planningWriteTool("Finalize planning workflow"),
	}, s.handleFinalizePlanningWorkflow)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "read_togpt_message",
		Description: "Reads root togpt.md, the timestamped response file written by the downstream implementation agent. Read-only.",
		Annotations: readOnlyTool("Read togpt.md"),
	}, s.handleReadToGPTMessage)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "append_fromgpt_message",
		Description: "Appends a timestamped planning-assistant message to root fromgpt.md for a downstream implementation agent. This tool writes only fromgpt.md.",
		Annotations: planningWriteTool("Append fromgpt.md"),
	}, s.handleAppendFromGPTMessage)
}

func readOnlyTool(title string) *mcp.ToolAnnotations {
	openWorld := false
	return &mcp.ToolAnnotations{
		Title:         title,
		ReadOnlyHint:  true,
		OpenWorldHint: &openWorld,
	}
}

func planningWriteTool(title string) *mcp.ToolAnnotations {
	destructive := false
	openWorld := false
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: &destructive,
		OpenWorldHint:   &openWorld,
	}
}

// ===== Handlers =====

func (s *Server) handleGetProjectBrainGuide(ctx context.Context, req *mcp.CallToolRequest, input GetProjectBrainGuideInput) (*mcp.CallToolResult, ProjectBrainGuideOutput, error) {
	if err := s.checkToolRate("get_project_brain_guide"); err != nil {
		s.audit("get_project_brain_guide", "", "", "blocked", err.Error(), 0)
		return errorResult[ProjectBrainGuideOutput](err.Error())
	}
	guide := plans.ProjectBrainGuideTemplate(input.Audience)
	out := ProjectBrainGuideOutput{
		Title:            "Project Brain MCP Context Summary",
		Version:          "1.1",
		Guide:            guide,
		RecommendedUsage: "Use this read-only context summary as background when discussing Project Brain MCP workflows.",
	}
	s.audit("get_project_brain_guide", "", "", "allowed", "", len(guide))
	return jsonContent(out)
}

func (s *Server) handleListRoots(ctx context.Context, req *mcp.CallToolRequest, _ ListRootsInput) (*mcp.CallToolResult, ListRootsOutput, error) {
	if err := s.checkToolRate("list_roots"); err != nil {
		s.audit("list_roots", "", "", "blocked", err.Error(), 0)
		return errorResult[ListRootsOutput](err.Error())
	}
	all := s.roots.All()
	var roots []RootInfo
	for _, r := range all {
		roots = append(roots, RootInfo{
			ID:       r.ID,
			Name:     r.Name,
			PathHint: r.Path,
			Mode:     s.cfg.Security.Mode,
		})
	}
	s.audit("list_roots", "", "", "allowed", "", 0)
	return jsonContent(ListRootsOutput{Roots: roots})
}

func (s *Server) handleListProjects(ctx context.Context, req *mcp.CallToolRequest, input ListProjectsInput) (*mcp.CallToolResult, ListProjectsOutput, error) {
	if err := s.checkToolRate("list_projects"); err != nil {
		s.audit("list_projects", input.RootID, "", "blocked", err.Error(), 0)
		return errorResult[ListProjectsOutput](err.Error())
	}
	if input.MaxDepth <= 0 {
		input.MaxDepth = 2
	}
	root, ok := s.roots.Get(input.RootID)
	if !ok {
		s.audit("list_projects", input.RootID, "", "blocked", "unknown root", 0)
		return errorResult[ListProjectsOutput](fmt.Sprintf("Unknown root: %s", input.RootID))
	}

	results, err := project.DetectProjects(root.Path, input.MaxDepth)
	if err != nil {
		s.audit("list_projects", input.RootID, "", "error", err.Error(), 0)
		return errorResult[ListProjectsOutput](fmt.Sprintf("Failed to list projects: %v", err))
	}

	var projects []ProjectInfo
	for _, r := range results {
		pid := fmt.Sprintf("%s:%s", input.RootID, r.RelativePath)
		projects = append(projects, ProjectInfo{
			ProjectID:     pid,
			Name:          r.Name,
			RelativePath:  r.RelativePath,
			DetectedStack: r.DetectedStack,
			HasGit:        r.HasGit,
		})
	}
	s.audit("list_projects", input.RootID, "", "allowed", "", 0)
	return jsonContent(ListProjectsOutput{Projects: projects})
}

func (s *Server) handleInspectProject(ctx context.Context, req *mcp.CallToolRequest, input InspectProjectInput) (*mcp.CallToolResult, InspectProjectOutput, error) {
	if err := s.checkToolRate("inspect_project"); err != nil {
		s.audit("inspect_project", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[InspectProjectOutput](err.Error())
	}
	_, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("inspect_project", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[InspectProjectOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	result, err := project.InspectProject(absPath, 200)
	if err != nil {
		s.audit("inspect_project", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[InspectProjectOutput](fmt.Sprintf("Inspection failed: %v", err))
	}

	if input.IncludeTree == nil || !*input.IncludeTree {
		result.TreePreview = nil
	}
	if input.IncludeManifests == nil || !*input.IncludeManifests {
		result.Manifests = nil
	}
	if input.IncludeGit == nil || !*input.IncludeGit {
		// Git info already limited in InspectProject.
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	resp := string(data)
	if s.cfg.Security.RedactSecrets {
		resp = fsx.RedactSecrets(resp)
		_ = json.Unmarshal([]byte(resp), result)
	}
	s.audit("inspect_project", input.ProjectID, "", "allowed", "", len(resp))
	return textContent(resp, InspectProjectOutput{Inspection: result})
}

func (s *Server) handleGetProjectTree(ctx context.Context, req *mcp.CallToolRequest, input GetProjectTreeInput) (*mcp.CallToolResult, GetProjectTreeOutput, error) {
	if err := s.checkToolRate("get_project_tree"); err != nil {
		s.audit("get_project_tree", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[GetProjectTreeOutput](err.Error())
	}
	if input.MaxEntries <= 0 {
		input.MaxEntries = 500
	}
	if input.Depth <= 0 {
		input.Depth = 4
	}
	_, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("get_project_tree", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[GetProjectTreeOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	tree, err := project.GetProjectTree(absPath, input.MaxEntries, input.Depth, s.cfg.Ignore.Dirs)
	if err != nil {
		s.audit("get_project_tree", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[GetProjectTreeOutput](fmt.Sprintf("Tree generation failed: %v", err))
	}
	s.audit("get_project_tree", input.ProjectID, "", "allowed", "", 0)
	return jsonContent(GetProjectTreeOutput{Entries: tree, Total: len(tree)})
}

func (s *Server) handleReadProjectFile(ctx context.Context, req *mcp.CallToolRequest, input ReadProjectFileInput) (*mcp.CallToolResult, ReadProjectFileOutput, error) {
	if err := s.checkToolRate("read_project_file"); err != nil {
		s.audit("read_project_file", input.ProjectID, input.Path, "blocked", err.Error(), 0)
		return errorResult[ReadProjectFileOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("read_project_file", input.ProjectID, input.Path, "blocked", err.Error(), 0)
		return errorResult[ReadProjectFileOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	rel := input.Path
	data, err := fsx.ReadFile(input.ProjectID, rel, absPath, root, s.guard, s.cfg.Security.MaxFileBytes)
	if err != nil {
		s.audit("read_project_file", input.ProjectID, input.Path, "blocked", err.Error(), 0)
		return errorResult[ReadProjectFileOutput](fmt.Sprintf("Read blocked: %v", err))
	}

	content := string(data)
	if s.cfg.Security.RedactSecrets {
		content = fsx.RedactSecrets(content)
	}
	s.audit("read_project_file", input.ProjectID, input.Path, "allowed", "", len(content))
	return textContent(content, ReadProjectFileOutput{ProjectID: input.ProjectID, Path: input.Path, Content: content})
}

func (s *Server) handleSearchProject(ctx context.Context, req *mcp.CallToolRequest, input SearchProjectInput) (*mcp.CallToolResult, SearchProjectOutput, error) {
	if err := s.checkToolRate("search_project"); err != nil {
		s.audit("search_project", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[SearchProjectOutput](err.Error())
	}
	if input.MaxResults <= 0 {
		input.MaxResults = s.cfg.Security.MaxSearchResults
		if input.MaxResults <= 0 {
			input.MaxResults = 100
		}
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("search_project", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[SearchProjectOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	results, err := fsx.SearchProject(input.ProjectID, absPath, root, s.guard, input.Query, input.Glob, input.MaxResults)
	if err != nil {
		s.audit("search_project", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[SearchProjectOutput](fmt.Sprintf("Search failed: %v", err))
	}
	s.audit("search_project", input.ProjectID, "", "allowed", "", 0)
	return jsonContent(SearchProjectOutput{Results: results, Total: len(results)})
}

func (s *Server) handleCreateAnalysisNote(ctx context.Context, req *mcp.CallToolRequest, input CreateAnalysisNoteInput) (*mcp.CallToolResult, WriteArtifactOutput, error) {
	if err := s.checkToolRate("create_project_analysis_note"); err != nil {
		s.audit("create_project_analysis_note", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("create_project_analysis_note", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	relPath, data, err := plans.Generate(input.ProjectID, plans.TypeAnalysis, input.Title, input.Content)
	if err != nil {
		s.audit("create_project_analysis_note", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Generation failed: %v", err))
	}

	if err := fsx.WriteFile(input.ProjectID, relPath, data, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("create_project_analysis_note", input.ProjectID, relPath, "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	s.audit("create_project_analysis_note", input.ProjectID, relPath, "allowed", "", len(data))
	return jsonContent(WriteArtifactOutput{WrittenTo: relPath, Status: "created"})
}

func (s *Server) handleCreateAgentPlan(ctx context.Context, req *mcp.CallToolRequest, input CreateAgentPlanInput) (*mcp.CallToolResult, WriteArtifactOutput, error) {
	if err := s.checkToolRate("create_agent_plan"); err != nil {
		s.audit("create_agent_plan", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("create_agent_plan", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	content := input.Content
	if strings.TrimSpace(content) == "" {
		content = plans.AgentPlanTemplate(input.Goal)
	}

	relPath, data, err := plans.Generate(input.ProjectID, plans.TypePlan, input.PlanTitle, content)
	if err != nil {
		s.audit("create_agent_plan", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Generation failed: %v", err))
	}

	if err := fsx.WriteFile(input.ProjectID, relPath, data, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("create_agent_plan", input.ProjectID, relPath, "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	s.audit("create_agent_plan", input.ProjectID, relPath, "allowed", "", len(data))
	return jsonContent(WriteArtifactOutput{WrittenTo: relPath, Status: "created"})
}

func (s *Server) handleCreateAgentHandoff(ctx context.Context, req *mcp.CallToolRequest, input CreateAgentHandoffInput) (*mcp.CallToolResult, WriteArtifactOutput, error) {
	if err := s.checkToolRate("create_agent_handoff"); err != nil {
		s.audit("create_agent_handoff", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("create_agent_handoff", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	relPath, data, err := plans.GenerateAgentHandoff(input.ProjectID, input.AgentName, input.TaskTitle, input.Content)
	if err != nil {
		s.audit("create_agent_handoff", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Generation failed: %v", err))
	}

	if err := fsx.WriteFile(input.ProjectID, relPath, data, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("create_agent_handoff", input.ProjectID, relPath, "blocked", err.Error(), 0)
		return errorResult[WriteArtifactOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	s.audit("create_agent_handoff", input.ProjectID, relPath, "allowed", "", len(data))
	return jsonContent(WriteArtifactOutput{WrittenTo: relPath, Status: "created"})
}

func (s *Server) handleBootstrapProjectAgentsMD(ctx context.Context, req *mcp.CallToolRequest, input BootstrapProjectAgentsInput) (*mcp.CallToolResult, BootstrapProjectAgentsOutput, error) {
	if err := s.checkToolRate("bootstrap_project_agents_md"); err != nil {
		s.audit("bootstrap_project_agents_md", input.ProjectID, "AGENTS.md", "blocked", err.Error(), 0)
		return errorResult[BootstrapProjectAgentsOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("bootstrap_project_agents_md", input.ProjectID, "AGENTS.md", "blocked", err.Error(), 0)
		return errorResult[BootstrapProjectAgentsOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	relPath := "AGENTS.md"
	target := filepath.Join(absPath, relPath)
	existed := false
	if _, err := os.Stat(target); err == nil {
		existed = true
		if !input.Overwrite {
			out := BootstrapProjectAgentsOutput{
				WrittenTo: relPath,
				Status:    "skipped_existing",
				NextSteps: "AGENTS.md already exists. Read it before deciding whether to call this tool again with overwrite set to true.",
			}
			s.audit("bootstrap_project_agents_md", input.ProjectID, relPath, "allowed", "existing file skipped", 0)
			return jsonContent(out)
		}
	} else if err != nil && !os.IsNotExist(err) {
		s.audit("bootstrap_project_agents_md", input.ProjectID, relPath, "error", err.Error(), 0)
		return errorResult[BootstrapProjectAgentsOutput](fmt.Sprintf("Cannot inspect AGENTS.md: %v", err))
	}

	data := []byte(plans.ProjectAgentsTemplate() + "\n")
	if err := fsx.WriteFile(input.ProjectID, relPath, data, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("bootstrap_project_agents_md", input.ProjectID, relPath, "blocked", err.Error(), 0)
		return errorResult[BootstrapProjectAgentsOutput](fmt.Sprintf("Write blocked: %v", err))
	}

	status := "created"
	if input.Overwrite && existed {
		status = "overwritten"
	}
	out := BootstrapProjectAgentsOutput{
		WrittenTo: relPath,
		Status:    status,
		NextSteps: "Reference AGENTS.md in implementation prompts so downstream agents understand the Project Brain MCP planning workflow.",
	}
	s.audit("bootstrap_project_agents_md", input.ProjectID, relPath, "allowed", "", len(data))
	return jsonContent(out)
}

func (s *Server) handleCreateQuickPlan(ctx context.Context, req *mcp.CallToolRequest, input CreateQuickPlanInput) (*mcp.CallToolResult, QuickPlanOutput, error) {
	if err := s.checkToolRate("create_quick_plan"); err != nil {
		s.audit("create_quick_plan", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[QuickPlanOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("create_quick_plan", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[QuickPlanOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	if root.ReadOnly {
		s.audit("create_quick_plan", input.ProjectID, "", "blocked", "root is read-only", 0)
		return errorResult[QuickPlanOutput]("Quick plan writes are blocked because this root is read-only")
	}
	if strings.TrimSpace(input.Objective) == "" {
		return errorResult[QuickPlanOutput]("objective is required")
	}
	content := plans.QuickPlanTemplate(plans.QuickPlanSpec{
		TaskTitle:          input.TaskTitle,
		Objective:          input.Objective,
		CurrentContext:     input.CurrentContext,
		RelevantFiles:      input.RelevantFiles,
		Phases:             input.Phases,
		AcceptanceCriteria: input.AcceptanceCriteria,
		Tests:              input.Tests,
		Risks:              input.Risks,
		Notes:              input.Notes,
	})
	planPath, planData, err := plans.Generate(input.ProjectID, plans.TypeQuickPlan, input.TaskTitle, content)
	if err != nil {
		s.audit("create_quick_plan", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[QuickPlanOutput](fmt.Sprintf("Generation failed: %v", err))
	}
	if err := fsx.WriteFile(input.ProjectID, planPath, planData, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("create_quick_plan", input.ProjectID, planPath, "blocked", err.Error(), 0)
		return errorResult[QuickPlanOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	out := QuickPlanOutput{
		PlanPath:     planPath,
		Status:       "created",
		ModeGuidance: "Use full planning workflow for product planning, architecture, migrations, auth/permissions, billing, or broad multi-module features.",
	}
	if input.CreateImplementationPrompt {
		promptContent := plans.ImplementationPromptTemplate(plans.ImplementationPromptSpec{
			TaskTitle:          input.TaskTitle,
			Objective:          input.Objective,
			PlanPath:           planPath,
			ContextFiles:       input.RelevantFiles,
			Constraints:        []string{"Keep the change scoped to the quick plan."},
			AcceptanceCriteria: input.AcceptanceCriteria,
			Notes:              input.Notes,
		})
		promptPath, promptData, err := plans.Generate(input.ProjectID, plans.TypeImplementationPrompt, input.TaskTitle, promptContent)
		if err != nil {
			return errorResult[QuickPlanOutput](fmt.Sprintf("Implementation prompt generation failed: %v", err))
		}
		if err := fsx.WriteFile(input.ProjectID, promptPath, promptData, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
			return errorResult[QuickPlanOutput](fmt.Sprintf("Implementation prompt write blocked: %v", err))
		}
		out.ImplementationPromptPath = promptPath
	}
	s.audit("create_quick_plan", input.ProjectID, planPath, "allowed", "", len(planData))
	return jsonContent(out)
}

func (s *Server) handleCreateImplementationPrompt(ctx context.Context, req *mcp.CallToolRequest, input CreateImplementationPromptInput) (*mcp.CallToolResult, ImplementationPromptOutput, error) {
	return s.createImplementationPrompt(input, "create_implementation_prompt", plans.TypeImplementationPrompt)
}

func (s *Server) handleCreateKimiPrompt(ctx context.Context, req *mcp.CallToolRequest, input CreateKimiPromptInput) (*mcp.CallToolResult, KimiPromptOutput, error) {
	return s.createImplementationPrompt(input, "create_kimi_prompt", plans.TypeKimi)
}

func (s *Server) createImplementationPrompt(input CreateImplementationPromptInput, toolName string, planType plans.PlanType) (*mcp.CallToolResult, ImplementationPromptOutput, error) {
	if err := s.checkToolRate(toolName); err != nil {
		s.audit(toolName, input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[ImplementationPromptOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit(toolName, input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[ImplementationPromptOutput](fmt.Sprintf("Invalid project: %v", err))
	}

	content := plans.ImplementationPromptTemplate(plans.ImplementationPromptSpec{
		TaskTitle:          input.TaskTitle,
		Objective:          input.Objective,
		PlanPath:           input.PlanPath,
		ContextFiles:       input.ContextFiles,
		Constraints:        input.Constraints,
		AcceptanceCriteria: input.AcceptanceCriteria,
		Notes:              input.Notes,
	})

	relPath, data, err := plans.Generate(input.ProjectID, planType, input.TaskTitle, content)
	if err != nil {
		s.audit(toolName, input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[ImplementationPromptOutput](fmt.Sprintf("Generation failed: %v", err))
	}

	if err := fsx.WriteFile(input.ProjectID, relPath, data, absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit(toolName, input.ProjectID, relPath, "blocked", err.Error(), 0)
		return errorResult[ImplementationPromptOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	s.audit(toolName, input.ProjectID, relPath, "allowed", "", len(data))
	return jsonContent(ImplementationPromptOutput{
		WrittenTo:       relPath,
		Status:          "created",
		CommandGuidance: "From the project root, pass this prompt file to your chosen implementation agent and instruct it to follow the file exactly.",
	})
}

func (s *Server) handleStartPlanningWorkflow(ctx context.Context, req *mcp.CallToolRequest, input StartPlanningWorkflowInput) (*mcp.CallToolResult, StartPlanningWorkflowOutput, error) {
	if err := s.checkToolRate("start_planning_workflow"); err != nil {
		s.audit("start_planning_workflow", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[StartPlanningWorkflowOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("start_planning_workflow", input.ProjectID, "", "blocked", err.Error(), 0)
		return errorResult[StartPlanningWorkflowOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	if root.ReadOnly {
		s.audit("start_planning_workflow", input.ProjectID, "", "blocked", "root is read-only", 0)
		return errorResult[StartPlanningWorkflowOutput]("Workflow writes are blocked because this root is read-only")
	}
	result, err := workflow.Start(absPath, input.ProjectID, input.Title, input.OriginalUserIntent, time.Now())
	if err != nil {
		s.audit("start_planning_workflow", input.ProjectID, "", "error", err.Error(), 0)
		return errorResult[StartPlanningWorkflowOutput](fmt.Sprintf("Workflow start failed: %v", err))
	}
	s.audit("start_planning_workflow", input.ProjectID, result.StatePath, "allowed", "", len(result.PhasePrompt))
	return jsonContent(StartPlanningWorkflowOutput{
		SessionID:      result.State.SessionID,
		StatePath:      result.StatePath,
		CurrentPhase:   result.State.CurrentPhase,
		PhasePrompt:    result.PhasePrompt,
		ArtifactTarget: result.ArtifactTarget,
		State:          result.State,
	})
}

func (s *Server) handleGetPlanningWorkflowStatus(ctx context.Context, req *mcp.CallToolRequest, input PlanningWorkflowSessionInput) (*mcp.CallToolResult, PlanningWorkflowStatusOutput, error) {
	if err := s.checkToolRate("get_planning_workflow_status"); err != nil {
		s.audit("get_planning_workflow_status", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[PlanningWorkflowStatusOutput](err.Error())
	}
	_, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("get_planning_workflow_status", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[PlanningWorkflowStatusOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	state, err := workflow.Load(absPath, input.SessionID)
	if err != nil {
		s.audit("get_planning_workflow_status", input.ProjectID, input.SessionID, "error", err.Error(), 0)
		return errorResult[PlanningWorkflowStatusOutput](fmt.Sprintf("Load workflow failed: %v", err))
	}
	s.audit("get_planning_workflow_status", input.ProjectID, input.SessionID, "allowed", "", 0)
	return jsonContent(PlanningWorkflowStatusOutput{State: state})
}

func (s *Server) handleGetCurrentPlanningPhase(ctx context.Context, req *mcp.CallToolRequest, input PlanningWorkflowSessionInput) (*mcp.CallToolResult, CurrentPlanningPhaseOutput, error) {
	if err := s.checkToolRate("get_current_planning_phase"); err != nil {
		s.audit("get_current_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CurrentPlanningPhaseOutput](err.Error())
	}
	_, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("get_current_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CurrentPlanningPhaseOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	state, err := workflow.Load(absPath, input.SessionID)
	if err != nil {
		s.audit("get_current_planning_phase", input.ProjectID, input.SessionID, "error", err.Error(), 0)
		return errorResult[CurrentPlanningPhaseOutput](fmt.Sprintf("Load workflow failed: %v", err))
	}
	phase, err := workflow.CurrentPhase(state)
	if err != nil {
		s.audit("get_current_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CurrentPlanningPhaseOutput](err.Error())
	}
	prompt := workflow.PhasePrompt(state, phase)
	s.audit("get_current_planning_phase", input.ProjectID, input.SessionID, "allowed", "", len(prompt))
	return jsonContent(CurrentPlanningPhaseOutput{
		SessionID:      state.SessionID,
		CurrentPhase:   phase,
		PhasePrompt:    prompt,
		ArtifactTarget: workflow.PhaseArtifactRelPath(state.SessionID, phase.ID, phase.Title),
		State:          state,
	})
}

func (s *Server) handleCompletePlanningPhase(ctx context.Context, req *mcp.CallToolRequest, input CompletePlanningPhaseInput) (*mcp.CallToolResult, CompletePlanningPhaseOutput, error) {
	if err := s.checkToolRate("complete_planning_phase"); err != nil {
		s.audit("complete_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CompletePlanningPhaseOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("complete_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CompletePlanningPhaseOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	if root.ReadOnly {
		s.audit("complete_planning_phase", input.ProjectID, input.SessionID, "blocked", "root is read-only", 0)
		return errorResult[CompletePlanningPhaseOutput]("Workflow writes are blocked because this root is read-only")
	}
	result, err := workflow.Complete(absPath, input.SessionID, input.PhaseID, input.Content, time.Now())
	if err != nil {
		s.audit("complete_planning_phase", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[CompletePlanningPhaseOutput](fmt.Sprintf("Complete phase failed: %v", err))
	}
	s.audit("complete_planning_phase", input.ProjectID, result.WrittenTo, "allowed", "", len(input.Content))
	phasePrompt := ""
	if result.NextPhase != nil {
		phasePrompt = workflow.PhasePrompt(result.State, *result.NextPhase)
	}
	return jsonContent(CompletePlanningPhaseOutput{
		SessionID:   result.State.SessionID,
		PhaseID:     input.PhaseID,
		WrittenTo:   result.WrittenTo,
		Status:      result.State.Status,
		NextPhase:   result.NextPhase,
		PhasePrompt: phasePrompt,
		NextAction:  result.NextAction,
		State:       result.State,
	})
}

func (s *Server) handleFinalizePlanningWorkflow(ctx context.Context, req *mcp.CallToolRequest, input FinalizePlanningWorkflowInput) (*mcp.CallToolResult, FinalizePlanningWorkflowOutput, error) {
	if err := s.checkToolRate("finalize_planning_workflow"); err != nil {
		s.audit("finalize_planning_workflow", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[FinalizePlanningWorkflowOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("finalize_planning_workflow", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[FinalizePlanningWorkflowOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	if root.ReadOnly {
		s.audit("finalize_planning_workflow", input.ProjectID, input.SessionID, "blocked", "root is read-only", 0)
		return errorResult[FinalizePlanningWorkflowOutput]("Workflow writes are blocked because this root is read-only")
	}
	result, err := workflow.Finalize(absPath, input.SessionID, input.MasterPlan, input.ImplementationPrompts, time.Now())
	if err != nil {
		s.audit("finalize_planning_workflow", input.ProjectID, input.SessionID, "blocked", err.Error(), 0)
		return errorResult[FinalizePlanningWorkflowOutput](fmt.Sprintf("Finalize workflow failed: %v", err))
	}
	s.audit("finalize_planning_workflow", input.ProjectID, result.MasterPlanPath, "allowed", "", len(input.MasterPlan))
	return jsonContent(FinalizePlanningWorkflowOutput{
		SessionID:             result.State.SessionID,
		MasterPlanPath:        result.MasterPlanPath,
		ImplementationPrompts: result.ImplementationPrompts,
		Status:                result.State.Status,
		State:                 result.State,
	})
}

func (s *Server) handleReadToGPTMessage(ctx context.Context, req *mcp.CallToolRequest, input ReadToGPTMessageInput) (*mcp.CallToolResult, ReadToGPTMessageOutput, error) {
	if err := s.checkToolRate("read_togpt_message"); err != nil {
		s.audit("read_togpt_message", input.ProjectID, "togpt.md", "blocked", err.Error(), 0)
		return errorResult[ReadToGPTMessageOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("read_togpt_message", input.ProjectID, "togpt.md", "blocked", err.Error(), 0)
		return errorResult[ReadToGPTMessageOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	data, err := fsx.ReadFile(input.ProjectID, "togpt.md", absPath, root, s.guard, s.cfg.Security.MaxFileBytes)
	if err != nil {
		if strings.Contains(err.Error(), "file not found") {
			s.audit("read_togpt_message", input.ProjectID, "togpt.md", "allowed", "missing", 0)
			return jsonContent(ReadToGPTMessageOutput{Path: "togpt.md", Status: "missing"})
		}
		s.audit("read_togpt_message", input.ProjectID, "togpt.md", "blocked", err.Error(), 0)
		return errorResult[ReadToGPTMessageOutput](fmt.Sprintf("Read blocked: %v", err))
	}
	content := string(data)
	if s.cfg.Security.RedactSecrets {
		content = fsx.RedactSecrets(content)
	}
	s.audit("read_togpt_message", input.ProjectID, "togpt.md", "allowed", "", len(content))
	return textContent(content, ReadToGPTMessageOutput{Path: "togpt.md", Status: "found", Content: content})
}

func (s *Server) handleAppendFromGPTMessage(ctx context.Context, req *mcp.CallToolRequest, input AppendFromGPTMessageInput) (*mcp.CallToolResult, AppendFromGPTMessageOutput, error) {
	if err := s.checkToolRate("append_fromgpt_message"); err != nil {
		s.audit("append_fromgpt_message", input.ProjectID, "fromgpt.md", "blocked", err.Error(), 0)
		return errorResult[AppendFromGPTMessageOutput](err.Error())
	}
	root, absPath, err := s.roots.ResolveProject(input.ProjectID)
	if err != nil {
		s.audit("append_fromgpt_message", input.ProjectID, "fromgpt.md", "blocked", err.Error(), 0)
		return errorResult[AppendFromGPTMessageOutput](fmt.Sprintf("Invalid project: %v", err))
	}
	if strings.TrimSpace(input.Message) == "" {
		return errorResult[AppendFromGPTMessageOutput]("message is required")
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Planning Assistant Message"
	}
	entry := fmt.Sprintf("## %s - %s\n\n%s\n", ts, title, strings.TrimSpace(input.Message))
	status := "created"
	existing, err := fsx.ReadFile(input.ProjectID, "fromgpt.md", absPath, root, s.guard, s.cfg.Security.MaxFileBytes)
	if err == nil && len(strings.TrimSpace(string(existing))) > 0 {
		entry = strings.TrimRight(string(existing), "\r\n") + "\n\n" + entry
		status = "appended"
	} else if err != nil && !strings.Contains(err.Error(), "file not found") {
		s.audit("append_fromgpt_message", input.ProjectID, "fromgpt.md", "blocked", err.Error(), 0)
		return errorResult[AppendFromGPTMessageOutput](fmt.Sprintf("Read existing fromgpt.md failed: %v", err))
	}
	if err := fsx.WriteFile(input.ProjectID, "fromgpt.md", []byte(entry), absPath, root, s.guard, s.cfg.Security.MaxFileBytes); err != nil {
		s.audit("append_fromgpt_message", input.ProjectID, "fromgpt.md", "blocked", err.Error(), 0)
		return errorResult[AppendFromGPTMessageOutput](fmt.Sprintf("Write blocked: %v", err))
	}
	s.audit("append_fromgpt_message", input.ProjectID, "fromgpt.md", "allowed", status, len(entry))
	return jsonContent(AppendFromGPTMessageOutput{WrittenTo: "fromgpt.md", Status: status, Timestamp: ts})
}
