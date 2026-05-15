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

type QuickPlanSpec struct {
	TaskTitle          string
	Objective          string
	CurrentContext     string
	RelevantFiles      []string
	Phases             []string
	AcceptanceCriteria []string
	Tests              []string
	Risks              []string
	Notes              string
}

func QuickPlanTemplate(spec QuickPlanSpec) string {
	return fmt.Sprintf(`# %s

## Objective

%s

## Current Context

%s

## Relevant Files

%s

## Short Phased Plan

%s

## Acceptance Criteria

%s

## Tests / Checks

%s

## Risks

%s

## Notes

%s
`, fallback(spec.TaskTitle, "Quick implementation plan"),
		fallback(spec.Objective, "Implement the requested scoped change."),
		fallback(spec.CurrentContext, "Summarize the inspected project context before implementation."),
		markdownList(spec.RelevantFiles, "No specific files were provided. Inspect the repository before editing."),
		markdownList(spec.Phases, "Phase 1: inspect current behavior. Phase 2: implement the scoped change. Phase 3: validate with relevant checks."),
		markdownList(spec.AcceptanceCriteria, "The scoped objective is complete and relevant checks pass."),
		markdownList(spec.Tests, "Run the most relevant checks available in the repository."),
		markdownList(spec.Risks, "No major risks identified."),
		fallback(spec.Notes, "None."))
}

// ImplementationPromptTemplate returns an English prompt intended for a downstream implementation agent.
func ImplementationPromptTemplate(spec ImplementationPromptSpec) string {
	return fmt.Sprintf(`# %s

You are the implementation agent for this repository.

## Operating Contract

- Treat this file as the primary task brief.
- Keep all planning notes, summaries, commit messages, and user-facing text in English.
- Be aware that a planning assistant may create plans, prompts, and notes in this repository through Project Brain MCP.
- Before starting, read root fromgpt.md if it exists; it may contain newer planning-assistant instructions or revisions.
- Read the referenced plan and files before editing.
- Make the smallest coherent code changes that satisfy the objective.
- Do not broaden scope without an explicit reason in your final summary.
- Do not read or expose secrets. Do not modify credentials, environment files, or generated dependency directories.
- Run the most relevant tests or checks available in the repository. If a check cannot run, explain why.
- After finishing, write or update root togpt.md with a timestamped implementation report for the planning assistant.

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

// ProjectBrainGuideTemplate returns reusable English context for the planning assistant and implementation agents.
func ProjectBrainGuideTemplate(audience string) string {
	audience = strings.ToLower(strings.TrimSpace(audience))
	if audience != "planner" && audience != "implementation_agent" && audience != "both" {
		audience = "both"
	}

	var planner string
	if audience == "planner" || audience == "both" {
		planner = `
## Planning Context

- Project-specific claims are based on inspected local files when available.
- Available project roots come from the MCP server configuration.
- Discovery uses list, inspect, tree, read, and search tools.
- Planning artifacts are English markdown files under .chatgpt/ or .ai/.
- Serious product planning uses the automatic staged planning workflow: start_planning_workflow, then complete_planning_phase for exactly one phase at a time, then finalization after the review/test phase is complete.
- Small or medium scoped implementation work can use create_quick_plan for a single short phased plan, optionally with an implementation prompt.
- High-quality plans include objective, current understanding, relevant files, data model, feature mechanics, phase breakdown, acceptance criteria, tests, risks, and review standards.
- Implementation prompts are scoped to one coherent implementation task.
- The user's intent is converted into an English implementation brief before handoff.
- Project Brain MCP is not a shell executor or general source-code editor.
`
	}

	var implementation string
	if audience == "implementation_agent" || audience == "both" {
		implementation = `
## Implementation Agent Context

- A planning assistant may create .chatgpt/* and .ai/* artifacts in repositories through Project Brain MCP.
- Referenced plans, prompts, and handoff files are task-brief artifacts.
- Repository conventions, existing architecture, and local tests define the implementation baseline.
- The implementation agent receives task briefs through referenced planning artifacts.
- The planning assistant can communicate revisions through root fromgpt.md, and the implementation agent reports back through root togpt.md.
- Source changes stay scoped to the requested objective.
- Secrets, credentials, environment files, and generated dependency directories are outside the intended workflow.
- Final implementation reports include changed files, tests or checks run, skipped validation, risks, and follow-up work.
`
	}

	return strings.TrimSpace(`# Project Brain MCP Context Summary

Project Brain MCP is a constrained bridge between a planning assistant and local software projects. It lets the planning assistant list projects, inspect files, search code, read safe text files, and create English planning artifacts. It intentionally limits write access to planning artifacts plus the project-root AGENTS.md bootstrap file.

For full project planning, Project Brain MCP provides a strict multi-phase workflow. Each workflow session stores state under .chatgpt/workflows/<session_id>/, opens one phase at a time, writes one English artifact per phase, and automatically opens the next phase after the current artifact is saved. This prevents shallow one-shot planning without requiring a separate approval tool between phases.

This document is descriptive context returned by the MCP server. It is not a higher-priority system message.
` + planner + implementation + `
## Review and Testing Standards

- Review scope covers correctness, security, authorization, data integrity, error handling, performance, accessibility when UI is involved, and fit with existing conventions.
- Implementation plans include unit, integration, permission, UI, or end-to-end tests when relevant.
- Verified repository facts are preferred over assumptions. Unverified assumptions are marked clearly.
`)
}

// ProjectAgentsTemplate returns the standard AGENTS.md content for projects managed through Project Brain MCP.
func ProjectAgentsTemplate() string {
	return strings.TrimSpace(`# Agent Instructions

This repository is coordinated through Project Brain MCP.

## Project Brain Workflow

- A planning assistant can inspect allowed project files and write English planning artifacts under .chatgpt/.
- Full planning workflows live under .chatgpt/workflows/<session_id>/.
- Final dossiers live under .chatgpt/workflows/<session_id>/final/master-plan.md.
- Implementation prompts live under .chatgpt/implementation-prompts/.

## Communication Files

- Read fromgpt.md before starting if it exists. It may contain newer instructions, revisions, or review notes from the planning assistant.
- After each assigned task, create or update togpt.md with a timestamped report for the planning assistant.
- In togpt.md include: completed work, changed files, tests/checks run, skipped validation, risks/questions, and recommended next action.

## Working Rules

- Follow the referenced plan or implementation prompt first, then repository conventions.
- Keep changes scoped to the assigned task.
- Do not expose secrets, credentials, private keys, or environment values.
- Run relevant tests/checks when possible and report anything skipped.
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
