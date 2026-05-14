package fsx

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"OPENAI_API_KEY=sk-abc123def45678901234567890",
			"OPENAI_API_KEY= [REDACTED]",
		},
		{
			"GITHUB_TOKEN=ghp_abcdefghijklmnopqrstuvwxyz12",
			"GITHUB_TOKEN= [REDACTED]",
		},
		{
			"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			"Authorization: [REDACTED]",
		},
		{
			"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			"AWS_ACCESS_KEY_ID= [REDACTED]",
		},
		{
			"password = mysecret12345",
			"password = [REDACTED]",
		},
		{
			"This is safe text without secrets",
			"This is safe text without secrets",
		},
	}

	for _, tt := range tests {
		got := RedactSecrets(tt.input)
		if !strings.Contains(got, "[REDACTED]") && strings.Contains(tt.expected, "[REDACTED]") {
			t.Errorf("RedactSecrets(%q) did not redact; got %q", tt.input, got)
		}
		if strings.Contains(got, "sk-abc123def4567890") || strings.Contains(got, "ghp_abcdefghijklmnopqrstuvwxyz12") {
			t.Errorf("RedactSecrets(%q) leaked secret; got %q", tt.input, got)
		}
	}
}
