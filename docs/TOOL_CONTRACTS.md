# Tool Contracts

This document defines the input/output contracts for every MCP tool exposed by Project Brain.

## `list_roots`

Lists configured project roots that this MCP server is allowed to access.

**Input:** `{}`

**Output:**
```json
{
  "roots": [
    {
      "id": "personal-projects",
      "name": "Personal Projects",
      "path_hint": "~/Projects",
      "mode": "planning_write"
    }
  ]
}
```

## `list_projects`

Lists software projects under a configured root. Does not read file contents.

**Input:**
```json
{
  "root_id": "personal-projects",
  "max_depth": 2
}
```

**Output:**
```json
{
  "projects": [
    {
      "project_id": "personal-projects:my-app",
      "name": "my-app",
      "relative_path": "my-app",
      "detected_stack": ["node", "nextjs", "react"],
      "has_git": true
    }
  ]
}
```

## `inspect_project`

Returns a structured read-only overview of a selected project.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "include_tree": true,
  "include_manifests": true,
  "include_git": true
}
```

**Output:**
```json
{
  "name": "my-app",
  "stack": ["nextjs", "typescript", "tailwind"],
  "entrypoints": ["app/page.tsx"],
  "package_managers": ["pnpm"],
  "important_files": ["package.json", "next.config.ts"],
  "summary": "Project \"my-app\" uses nextjs, typescript, tailwind.",
  "tree_preview": ["app/", "app/page.tsx", ...],
  "manifests": { "package.json": { ... } },
  "warnings": [".env file present in project root"]
}
```

## `get_project_tree`

Returns a filtered file tree for a project, respecting ignore rules.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "max_entries": 500,
  "depth": 4
}
```

**Output:**
```json
{
  "entries": [
    { "path": "app", "is_dir": true },
    { "path": "app/page.tsx", "is_dir": false, "size": 1234 }
  ],
  "total": 2
}
```

## `read_project_file`

Reads the contents of a single allowed file from a project.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "path": "package.json"
}
```

**Output:** File contents as text. Errors for:
- File too large (> max_file_bytes)
- Binary files
- Sensitive files
- Path traversal

## `search_project`

Searches for a query string across text files in a project.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "query": "auth middleware",
  "glob": "**/*.{ts,tsx}",
  "max_results": 50
}
```

**Output:**
```json
{
  "results": [
    {
      "path": "src/auth.ts",
      "line": 42,
      "content": "export function authMiddleware()"
    }
  ],
  "total": 1
}
```

## `get_project_brain_guide`

Returns the reusable English Project Brain MCP operating guide.

**Input:**
```json
{
  "audience": "both"
}
```

`audience` may be `planner`, `implementation_agent`, or `both`.

**Output:**
```json
{
  "title": "Project Brain MCP Context Summary",
  "version": "1.0",
  "guide": "...",
  "recommended_usage": "..."
}
```

## `bootstrap_project_agents_md`

Creates or optionally overwrites a project-root `AGENTS.md` explaining the Project Brain MCP workflow to downstream implementation agents.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "overwrite": false
}
```

**Output:**
```json
{
  "written_to": "AGENTS.md",
  "status": "created",
  "next_steps": "Reference AGENTS.md in implementation prompts so downstream agents understand the Project Brain MCP planning workflow."
}
```

If `AGENTS.md` already exists and `overwrite` is false, the status is `skipped_existing`.

Freeform planning and implementation-prompt tools are intentionally not exposed. Scoped small/medium implementation work may use `create_quick_plan`. Serious planning must use `start_planning_workflow`, `complete_planning_phase`, explicit approval, and `finalize_planning_workflow`.

## `create_quick_plan`

Creates one English short phased implementation plan for small or medium scoped work after project inspection. Use this for bug fixes, small UI changes, endpoint additions, focused refactors, or test additions.

Do not use it for product planning, architecture decisions, migrations, auth/permission changes, billing/payment work, production-critical refactors, or broad multi-module features. Use the full planning workflow instead.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "task_title": "Fix dashboard empty state",
  "objective": "Improve the dashboard empty state without changing data loading behavior.",
  "current_context": "Summary of inspected files, current patterns, and risk areas.",
  "relevant_files": ["app/dashboard/page.tsx"],
  "phases": ["P1.1 Inspect current rendering.", "P1.2 Implement the scoped UI change.", "P1.3 Validate checks."],
  "acceptance_criteria": ["The empty state renders clearly on mobile and desktop."],
  "tests": ["Run the relevant frontend checks."],
  "risks": ["Do not alter data fetching behavior."],
  "notes": "Optional extra context.",
  "create_implementation_prompt": true
}
```

**Output:**
```json
{
  "plan_path": ".chatgpt/quick-plans/20260515-010000-fix-dashboard-empty-state.md",
  "status": "created",
  "implementation_prompt_path": ".chatgpt/implementation-prompts/20260515-010000-fix-dashboard-empty-state.md",
  "mode_guidance": "Use full planning workflow for product planning, architecture, migrations, auth/permissions, billing, or broad multi-module features."
}
```

## `start_planning_workflow`

Starts a strict manual-gated planning workflow under `.chatgpt/workflows/<session_id>/`.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "title": "social-app",
  "original_user_intent": "Raw user idea. It may be Turkish; artifacts must be English."
}
```

**Output:**
```json
{
  "session_id": "20260514-080000-social-app",
  "state_path": ".chatgpt/workflows/20260514-080000-social-app/workflow.json",
  "current_phase": "01-intent",
  "phase_prompt": "...",
  "artifact_target": ".chatgpt/workflows/20260514-080000-social-app/phases/01-intent-intent.md",
  "state": { "...": "..." }
}
```

## `get_planning_workflow_status`

Returns the workflow state. Read-only.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "session_id": "20260514-080000-social-app"
}
```

**Output:**
```json
{
  "state": { "...": "..." }
}
```

## `get_current_planning_phase`

Returns the active phase contract, prompt, quality checklist, and target artifact path. Read-only.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "session_id": "20260514-080000-social-app"
}
```

**Output:**
```json
{
  "session_id": "20260514-080000-social-app",
  "current_phase": { "id": "01-intent", "title": "Intent" },
  "phase_prompt": "...",
  "artifact_target": ".chatgpt/workflows/20260514-080000-social-app/phases/01-intent-intent.md",
  "state": { "...": "..." }
}
```

## `complete_planning_phase`

Writes the current phase artifact and sets the phase to `awaiting_user_approval`. This tool does not advance to the next phase.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "session_id": "20260514-080000-social-app",
  "phase_id": "01-intent",
  "content": "# Intent\n\n..."
}
```

**Output:**
```json
{
  "session_id": "20260514-080000-social-app",
  "phase_id": "01-intent",
  "written_to": ".chatgpt/workflows/20260514-080000-social-app/phases/01-intent-intent.md",
  "status": "awaiting_user_approval",
  "next_action": "Review this phase artifact with the user...",
  "state": { "...": "..." }
}
```

## `approve_planning_phase`

Advances the workflow only after explicit user approval.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "session_id": "20260514-080000-social-app",
  "phase_id": "01-intent"
}
```

**Output:**
```json
{
  "session_id": "20260514-080000-social-app",
  "approved_phase": "01-intent",
  "status": "in_progress",
  "next_phase": { "id": "02-deep-search", "title": "Deep Search" },
  "phase_prompt": "...",
  "next_action": "The next phase is now open...",
  "state": { "...": "..." }
}
```

## `finalize_planning_workflow`

Writes the final master dossier and implementation prompts after all ten phases are approved.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "session_id": "20260514-080000-social-app",
  "master_plan": "# Master Plan\n\n...",
  "implementation_prompts": [
    {
      "title": "P1.1 Auth",
      "objective": "Implement the P1.1 slice.",
      "acceptance_criteria": ["Relevant checks pass."]
    }
  ]
}
```

**Output:**
```json
{
  "session_id": "20260514-080000-social-app",
  "master_plan_path": ".chatgpt/workflows/20260514-080000-social-app/final/master-plan.md",
  "implementation_prompts": [".chatgpt/implementation-prompts/2026-05-14-social-app-p1-1-auth.md"],
  "status": "completed",
  "state": { "...": "..." }
}
```

## `read_togpt_message`

Reads root `togpt.md`, the timestamped response file written by a downstream implementation agent.

**Input:**
```json
{
  "project_id": "personal-projects:my-app"
}
```

**Output when present:**
```json
{
  "path": "togpt.md",
  "status": "found",
  "content": "## 2026-05-14T09:00:00Z\n..."
}
```

**Output when missing:**
```json
{
  "path": "togpt.md",
  "status": "missing",
  "content": ""
}
```

## `append_fromgpt_message`

Appends a timestamped planning-assistant message to root `fromgpt.md` for the downstream implementation agent.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "title": "Revision",
  "message": "Please revise P1.1 according to the updated review notes."
}
```

**Output:**
```json
{
  "written_to": "fromgpt.md",
  "status": "created",
  "timestamp": "2026-05-14T09:00:00Z"
}
```

## Error Handling

All tools return MCP tool errors (not protocol errors) for business-logic failures:

- `isError: true` with descriptive text for blocked/disallowed operations
- Standard JSON-RPC errors for protocol issues
