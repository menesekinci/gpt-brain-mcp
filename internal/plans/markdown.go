package plans

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// PlanType defines the kind of markdown file.
type PlanType string

const (
	TypeAnalysis             PlanType = "analysis"
	TypePlan                 PlanType = "plan"
	TypeHandoff              PlanType = "handoff"
	TypeImplementationPrompt PlanType = "implementation_prompt"
	TypeKimi                 PlanType = "kimi_prompt"
	TypeDecision             PlanType = "decision"
)

// Generate creates a markdown file with frontmatter.
func Generate(projectID string, planType PlanType, title string, content string) (string, []byte, error) {
	safeTitle := slugify(title)
	datePrefix := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", datePrefix, safeTitle)

	var subdir string
	switch planType {
	case TypeAnalysis:
		subdir = "analysis"
	case TypePlan:
		subdir = "plans"
	case TypeHandoff:
		subdir = "handoffs"
	case TypeImplementationPrompt:
		subdir = "implementation-prompts"
	case TypeKimi:
		subdir = "kimi"
	case TypeDecision:
		subdir = "decisions"
	default:
		subdir = "notes"
	}

	relPath := filepath.ToSlash(filepath.Join(".chatgpt", subdir, filename))
	if planType == TypeHandoff {
		// handoffs may include agent name in filename.
		relPath = filepath.ToSlash(filepath.Join(".chatgpt", subdir, filename))
	}

	frontmatter := fmt.Sprintf(`---
type: %s
id: %s
created_at: %s
project_id: %s
status: draft
mode: planning_write
generated_by: chatgpt
---

`, planType, safeTitle, time.Now().UTC().Format(time.RFC3339), projectID)

	fullContent := frontmatter + strings.TrimSpace(content) + "\n"
	return relPath, []byte(fullContent), nil
}

// GenerateAgentHandoff creates a handoff file with agent name in filename.
func GenerateAgentHandoff(projectID string, agentName string, taskTitle string, content string) (string, []byte, error) {
	safeTitle := slugify(taskTitle)
	datePrefix := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s-%s.md", datePrefix, slugify(agentName), safeTitle)
	relPath := filepath.ToSlash(filepath.Join(".chatgpt", "handoffs", filename))

	frontmatter := fmt.Sprintf(`---
type: agent_handoff
id: %s
created_at: %s
project_id: %s
agent: %s
status: draft
mode: planning_write
generated_by: chatgpt
---

`, safeTitle, time.Now().UTC().Format(time.RFC3339), projectID, agentName)

	fullContent := frontmatter + strings.TrimSpace(content) + "\n"
	return relPath, []byte(fullContent), nil
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
