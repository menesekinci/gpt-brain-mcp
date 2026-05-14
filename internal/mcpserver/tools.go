package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enes/project-brain-mcp/internal/fsx"
	"github.com/enes/project-brain-mcp/internal/plans"
	"github.com/enes/project-brain-mcp/internal/project"
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

// ===== Tool Registration =====

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_project_brain_guide",
		Description: "Use this when a ChatGPT conversation needs the reusable Project Brain MCP operating guide. Returns English instructions for the planning assistant and/or downstream implementation agents. This tool does not read project files or write anything.",
	}, s.handleGetProjectBrainGuide)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_roots",
		Description: "Lists configured project roots that this MCP server is allowed to access.",
	}, s.handleListRoots)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_projects",
		Description: "Lists software projects under a configured root. Does not read file contents.",
	}, s.handleListProjects)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "inspect_project",
		Description: "Returns a structured read-only overview of a selected project, including detected stack, important files, manifest summaries, and warnings.",
	}, s.handleInspectProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_project_tree",
		Description: "Returns a filtered file tree for a project, respecting ignore rules and depth limits.",
	}, s.handleGetProjectTree)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "read_project_file",
		Description: "Reads the contents of a single allowed file from a project. Binary files, secrets, and oversized files are rejected.",
	}, s.handleReadProjectFile)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_project",
		Description: "Searches for a query string across text files in a project. Returns matching lines with file paths and line numbers.",
	}, s.handleSearchProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_project_analysis_note",
		Description: "Creates a markdown analysis note inside the selected project's .chatgpt/analysis directory. This tool cannot write outside the configured planning directories.",
	}, s.handleCreateAnalysisNote)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_agent_plan",
		Description: "Creates a markdown implementation plan inside the selected project's .chatgpt/plans directory. This tool cannot write outside the configured planning directories.",
	}, s.handleCreateAgentPlan)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_agent_handoff",
		Description: "Creates a markdown handoff file inside the selected project's .chatgpt/handoffs directory for an external coding agent. This tool cannot write outside the configured planning directories.",
	}, s.handleCreateAgentHandoff)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "bootstrap_project_agents_md",
		Description: "Use this once per project to create a standard English project-root AGENTS.md explaining the Project Brain MCP workflow to downstream implementation agents. This tool writes only AGENTS.md and does not modify source code.",
	}, s.handleBootstrapProjectAgentsMD)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_implementation_prompt",
		Description: "Use this when ChatGPT has planned a coding task and needs to hand implementation to a downstream implementation agent. Creates an English markdown prompt under .chatgpt/implementation-prompts. This tool does not execute agents or modify source files.",
	}, s.handleCreateImplementationPrompt)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_kimi_prompt",
		Description: "Compatibility alias for older workflows. Prefer create_implementation_prompt for new tasks. Creates a generic English implementation prompt without assuming a specific implementation agent, model, IDE, or CLI.",
	}, s.handleCreateKimiPrompt)
}

// ===== Handlers =====

func (s *Server) handleGetProjectBrainGuide(ctx context.Context, req *mcp.CallToolRequest, input GetProjectBrainGuideInput) (*mcp.CallToolResult, ProjectBrainGuideOutput, error) {
	if err := s.checkToolRate("get_project_brain_guide"); err != nil {
		s.audit("get_project_brain_guide", "", "", "blocked", err.Error(), 0)
		return errorResult[ProjectBrainGuideOutput](err.Error())
	}
	guide := plans.ProjectBrainGuideTemplate(input.Audience)
	out := ProjectBrainGuideOutput{
		Title:            "Project Brain MCP Operating Guide",
		Version:          "1.0",
		Guide:            guide,
		RecommendedUsage: "Paste or reference this guide at the start of normal ChatGPT conversations that should use Project Brain MCP as the project planning and handoff context.",
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
