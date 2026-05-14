package fsx

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)OPENAI_API_KEY\s*[:=]\s*["']?[a-zA-Z0-9_\-]{20,}["']?`),
	regexp.MustCompile(`(?i)GITHUB_TOKEN\s*[:=]\s*["']?[a-zA-Z0-9_\-]{20,}["']?`),
	regexp.MustCompile(`(?i)CLOUDFLARE_API_TOKEN\s*[:=]\s*["']?[a-zA-Z0-9_\-]{20,}["']?`),
	regexp.MustCompile(`(?i)AWS_ACCESS_KEY_ID\s*[:=]\s*["']?[A-Z0-9]{20}["']?`),
	regexp.MustCompile(`(?i)AWS_SECRET_ACCESS_KEY\s*[:=]\s*["']?[a-zA-Z0-9/+=]{40}["']?`),
	regexp.MustCompile(`(?i)PRIVATE\s+KEY.*-----END`),
	regexp.MustCompile(`(?i)Bearer\s+[a-zA-Z0-9_\-\.]{20,}`),
	regexp.MustCompile(`(?i)sk-[a-zA-Z0-9]{20,}`),
	regexp.MustCompile(`(?i)ghp_[a-zA-Z0-9]{30,}`),
	regexp.MustCompile(`(?i)gho_[a-zA-Z0-9]{30,}`),
	regexp.MustCompile(`(?i)glpat-[a-zA-Z0-9_\-]{20,}`),
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*["']?[a-zA-Z0-9_\-]{16,}["']?`),
	regexp.MustCompile(`(?i)password\s*[:=]\s*["']?[^\s"']{8,}["']?`),
	regexp.MustCompile(`(?i)secret\s*[:=]\s*["']?[^\s"']{8,}["']?`),
	regexp.MustCompile(`(?i)token\s*[:=]\s*["']?[a-zA-Z0-9_\-\.]{16,}["']?`),
}

// RedactSecrets replaces likely secrets with [REDACTED].
func RedactSecrets(input string) string {
	for _, re := range secretPatterns {
		input = re.ReplaceAllStringFunc(input, func(match string) string {
			// Find the separator position to preserve key name if possible.
			idx := strings.IndexAny(match, ":=")
			if idx > 0 {
				return match[:idx+1] + " [REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return input
}
