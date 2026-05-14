# Project Brain MCP Agent Instructions

## Purpose

Project Brain MCP lets a planning assistant inspect local project files through a remote MCP endpoint and write English planning artifacts for downstream coding agents.

## Required Workflow

- Use the planning assistant for project understanding, planning, prompt writing, and review.
- Keep every planning-assistant-authored plan, handoff, analysis note, and implementation prompt in English.
- Use a downstream implementation agent by giving it a referenced prompt file from `.chatgpt/implementation-prompts/`.
- Do not ask implementation agents to infer the user's original intent directly when an English plan or prompt can be provided.
- Keep MCP server write access limited to planning artifacts under `.chatgpt/` or `.ai/`, plus the exact project-root `AGENTS.md` and `fromgpt.md` files.
- Do not add a generic command-execution MCP tool unless the security model is redesigned and explicitly approved.
- When a planning assistant needs to send a message, revision, or follow-up to an implementation agent, it should append a timestamped entry to root `fromgpt.md`.
- When an implementation agent finishes a task, it must create or update root `togpt.md` with a timestamped response for the planning assistant.
- `togpt.md` should include completed work, changed files, tests/checks run, skipped validation, risks, questions, and recommended next action.

## Security Rules

- Never expose secrets, credentials, private keys, or environment files.
- Never put secrets, credentials, private keys, or environment values in `fromgpt.md` or `togpt.md`.
- Never broaden filesystem access beyond configured roots.
- Prefer read-only inspection plus markdown planning writes.
- If a generated prompt asks an implementation agent to run commands, it must also ask the agent to report what it ran.
