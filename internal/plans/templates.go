package plans

import (
	"fmt"
	"strings"
)

// AgentPlanTemplate returns a default agent plan markdown body.
func AgentPlanTemplate(goal string) string {
	return fmt.Sprintf(`# %s

## Goal

%s

## Current Understanding

- Describe what you understand about the current codebase.
- Note any assumptions or open questions.

## Relevant Files

- List files relevant to this plan.

## Constraints

- Do not change public API unless necessary.
- Preserve existing behavior.
- Add tests before risky changes.

## Proposed Phases

### Phase 1 — Discovery

- [ ] Read current implementation.
- [ ] Document current behavior.

### Phase 2 — Implementation

- [ ] Make changes.
- [ ] Add unit tests.

### Phase 3 — Validation

- [ ] Run tests.
- [ ] Run lint.
- [ ] Manual smoke test.

## Acceptance Criteria

- [ ] Existing functionality still works.
- [ ] New tests added.
- [ ] No secrets added to repo.

## Risks

- List risks and mitigation strategies.

## Instructions for Coding Agent

- This plan must be written and maintained in English.
- Provide step-by-step instructions for the agent.
`, goal, goal)
}

// ImplementationPromptSpec contains the structured inputs for an implementation agent prompt.
type ImplementationPromptSpec struct {
	TaskTitle          string
	Objective          string
	PlanPath           string
	ContextFiles       []string
	Constraints        []string
	AcceptanceCriteria []string
	Notes              string
}

// ImplementationPromptTemplate returns an English prompt intended for a downstream implementation agent.
func ImplementationPromptTemplate(spec ImplementationPromptSpec) string {
	return fmt.Sprintf(`# %s

You are the implementation agent for this repository.

## Operating Contract

- Treat this file as the primary task brief.
- Keep all planning notes, summaries, commit messages, and user-facing text in English.
- Be aware that a planning assistant may create plans, prompts, and notes in this repository through Project Brain MCP.
- Read the referenced plan and files before editing.
- Make the smallest coherent code changes that satisfy the objective.
- Do not broaden scope without an explicit reason in your final summary.
- Do not read or expose secrets. Do not modify credentials, environment files, or generated dependency directories.
- Run the most relevant tests or checks available in the repository. If a check cannot run, explain why.

## Objective

%s

## Source Plan

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

Report:

- Files changed
- Tests or checks run
- Any skipped validation and why
- Remaining risks or follow-up work
`, fallback(spec.TaskTitle, "Implementation task"),
		fallback(spec.Objective, "Implement the requested change according to the referenced plan."),
		fallback(spec.PlanPath, "No separate source plan was provided. Use this prompt as the source plan."),
		markdownList(spec.ContextFiles, "No specific context files were provided. Inspect the repository before editing."),
		markdownList(spec.Constraints, "Follow the repository's existing conventions and keep the change scoped."),
		markdownList(spec.AcceptanceCriteria, "The objective is implemented and relevant checks pass."),
		fallback(spec.Notes, "None."))
}

// KimiPromptSpec is kept as a compatibility alias for older callers.
type KimiPromptSpec = ImplementationPromptSpec

// KimiPromptTemplate is kept as a compatibility wrapper for older callers.
func KimiPromptTemplate(spec KimiPromptSpec) string {
	return ImplementationPromptTemplate(ImplementationPromptSpec(spec))
}

// ProjectBrainGuideTemplate returns reusable English operating guidance for the planning assistant and implementation agents.
func ProjectBrainGuideTemplate(audience string) string {
	audience = strings.ToLower(strings.TrimSpace(audience))
	if audience != "planner" && audience != "implementation_agent" && audience != "both" {
		audience = "both"
	}

	var planner string
	if audience == "planner" || audience == "both" {
		planner = `
## Planning Assistant Guidance

- Use Project Brain MCP to inspect local project files before making project-specific claims.
- Available project roots are controlled by the MCP server configuration; do not assume access outside listed roots.
- Use read/search/inspect tools for discovery, then write English planning artifacts under .chatgpt/ or .ai/.
- Keep plans specific: objective, current understanding, relevant files, data model, feature mechanics, phase breakdown, acceptance criteria, tests, risks, and review standards.
- Create one implementation prompt per coherent implementation task.
- Do not ask a downstream implementation agent to infer the user's original intent; convert it into an English implementation brief first.
- Project Brain MCP should not be used as a generic shell executor or source-code editor.
`
	}

	var implementation string
	if audience == "implementation_agent" || audience == "both" {
		implementation = `
## Implementation Agent Guidance

- A planning assistant may create .chatgpt/* and .ai/* artifacts in this repository through Project Brain MCP.
- Treat referenced plans, prompts, and handoff files as task briefs.
- Read the referenced files before editing and follow the repository's existing conventions.
- Keep changes scoped to the requested objective.
- Do not read, expose, or modify secrets, credentials, environment files, or generated dependency directories.
- Run the most relevant tests or checks available. If validation cannot run, explain why.
- Final responses should report changed files, tests/checks run, skipped validation, risks, and follow-up work.
`
	}

	return strings.TrimSpace(`# Project Brain MCP Operating Guide

Project Brain MCP is a constrained bridge between a planning assistant and local software projects. It lets the planning assistant list projects, inspect files, search code, read safe text files, and create English planning artifacts. It intentionally limits write access to planning artifacts plus the project-root AGENTS.md bootstrap file.

Use this guide to restore the operating context in normal ChatGPT conversations when a custom GPT cannot directly carry the MCP app configuration.
` + planner + implementation + `
## Review and Testing Standards

- Review for correctness, security, authorization, data integrity, error handling, performance, accessibility when UI is involved, and fit with existing conventions.
- Every implementation plan should include unit, integration, permission, UI or end-to-end tests when relevant.
- Prefer verified repository facts over assumptions. Mark any unverified assumption clearly.
`)
}

// ProjectAgentsTemplate returns the standard AGENTS.md content for projects managed through Project Brain MCP.
func ProjectAgentsTemplate() string {
	return strings.TrimSpace(`# Agent Instructions

This repository may be planned through Project Brain MCP.

Project Brain MCP is a constrained planning bridge. A planning assistant can inspect allowed project files and create English planning artifacts such as plans, prompts, analysis notes, and handoffs under .chatgpt/ or .ai/. The planning assistant may also maintain this AGENTS.md file to explain the workflow to downstream agents.

## How to Work

- Read the referenced plan, prompt, or handoff before editing.
- Treat planning artifacts as the task brief unless the user gives a newer instruction.
- Follow the repository's existing architecture, style, naming, and test conventions.
- Keep changes scoped to the requested objective.
- Do not broaden the task without explaining the reason in your final response.
- Do not read, expose, or modify secrets, credentials, environment files, or generated dependency directories.
- Keep planning notes, summaries, commit messages, and user-facing implementation reports in English unless the user explicitly requests otherwise.

## Validation

- Run the most relevant tests or checks available in this repository.
- If a check cannot run, explain what was skipped and why.
- Report changed files, tests/checks run, skipped validation, remaining risks, and follow-up work.
`)
}

func markdownList(items []string, empty string) string {
	var lines []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			lines = append(lines, "- "+item)
		}
	}
	if len(lines) == 0 {
		return "- " + empty
	}
	return strings.Join(lines, "\n")
}

func fallback(value, empty string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return empty
	}
	return value
}

// AnalysisNoteTemplate returns a default analysis markdown body.
func AnalysisNoteTemplate(projectName string) string {
	return fmt.Sprintf(`# Architecture Review: %s

## Overview

Brief description of the project.

## Stack

Detected technologies and frameworks.

## Structure

Key directories and their purposes.

## Entrypoints

Main entry points of the application.

## Dependencies

Notable dependencies and their roles.

## Warnings

Potential issues or technical debt.

## Recommendations

Suggested improvements or next steps.
`, projectName)
}
