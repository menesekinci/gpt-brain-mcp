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

## `create_project_analysis_note`

Creates a markdown analysis note inside `.chatgpt/analysis/`.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "title": "initial-architecture-review",
  "content": "# Architecture Review\n..."
}
```

**Output:**
```json
{
  "written_to": ".chatgpt/analysis/2026-05-13-initial-architecture-review.md",
  "status": "created"
}
```

## `create_agent_plan`

Creates a markdown implementation plan inside `.chatgpt/plans/`.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "plan_title": "auth-refactor-plan",
  "goal": "Refactor auth middleware",
  "content": "..."
}
```

If `content` is empty, a template is generated automatically.

**Output:**
```json
{
  "written_to": ".chatgpt/plans/2026-05-13-auth-refactor-plan.md",
  "status": "created"
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
  "title": "Project Brain MCP Operating Guide",
  "version": "1.0",
  "guide": "...",
  "recommended_usage": "..."
}
```

## `create_agent_handoff`

Creates a markdown handoff file inside `.chatgpt/handoffs/`.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "agent_name": "codex",
  "task_title": "implement-auth-refactor",
  "content": "..."
}
```

**Output:**
```json
{
  "written_to": ".chatgpt/handoffs/2026-05-13-codex-implement-auth-refactor.md",
  "status": "created"
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

## `create_implementation_prompt`

Creates an English markdown implementation prompt for a downstream implementation agent inside `.chatgpt/implementation-prompts/`.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "task_title": "implement-auth-refactor",
  "objective": "Implement the authentication refactor described in the source plan.",
  "plan_path": ".chatgpt/plans/2026-05-14-auth-refactor.md",
  "context_files": ["internal/auth/token.go", "internal/auth/oauth_server.go"],
  "constraints": ["Keep the public API stable."],
  "acceptance_criteria": ["go test ./... passes."],
  "notes": "Prefer small, reviewable changes."
}
```

**Output:**
```json
{
  "written_to": ".chatgpt/implementation-prompts/2026-05-14-implement-auth-refactor.md",
  "status": "created",
  "command_guidance": "From the project root, pass this prompt file to your chosen implementation agent and instruct it to follow the file exactly."
}
```

## `create_kimi_prompt`

Compatibility alias for older workflows. Prefer `create_implementation_prompt` for new tasks. The generated content is generic and does not assume a specific implementation agent, model, IDE, or CLI.

**Input:**
```json
{
  "project_id": "personal-projects:my-app",
  "task_title": "implement-auth-refactor",
  "objective": "Implement the authentication refactor described in the source plan.",
  "plan_path": ".chatgpt/plans/2026-05-14-auth-refactor.md",
  "context_files": ["internal/auth/token.go", "internal/auth/oauth_server.go"],
  "constraints": ["Keep the public API stable."],
  "acceptance_criteria": ["go test ./... passes."],
  "notes": "Prefer small, reviewable changes."
}
```

**Output:**
```json
{
  "written_to": ".chatgpt/kimi/2026-05-14-implement-auth-refactor.md",
  "status": "created",
  "command_guidance": "From the project root, pass this prompt file to your chosen implementation agent and instruct it to follow the file exactly."
}
```

## Error Handling

All tools return MCP tool errors (not protocol errors) for business-logic failures:

- `isError: true` with descriptive text for blocked/disallowed operations
- Standard JSON-RPC errors for protocol issues
