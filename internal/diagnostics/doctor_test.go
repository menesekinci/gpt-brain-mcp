package diagnostics

import (
	"strings"
	"testing"
)

func TestExplainPublicHTTPFailure(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "cloudflare 502",
			status: 502,
			body:   "Bad Gateway",
			want:   "HTTP 502",
		},
		{
			name:   "cloudflare 1033",
			status: 530,
			body:   "error code: 1033",
			want:   "no active tunnel connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := explainPublicHTTPFailure(tt.status, tt.body)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("expected %q to contain %q", got, tt.want)
			}
		})
	}
}

func TestFormatShowsFailures(t *testing.T) {
	report := Report{Checks: []Check{
		{Name: "Local MCP server", Status: StatusPass, Detail: "ok"},
		{Name: "Public connector endpoint", Status: StatusFail, Detail: "HTTP 502", Fix: "restart tunnel"},
	}}
	out := Format(report)
	for _, want := range []string{"Project Brain Doctor", "[OK] Local MCP server", "[FAIL] Public connector endpoint", "restart tunnel", "one or more checks failed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected formatted report to contain %q, got %s", want, out)
		}
	}
}
