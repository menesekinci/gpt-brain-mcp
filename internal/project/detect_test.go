package project

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDetectProjects(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "app-a", "src"), 0o755)
	os.WriteFile(filepath.Join(root, "app-a", "package.json"), []byte(`{"name":"app-a"}`), 0o644)

	os.MkdirAll(filepath.Join(root, "app-b"), 0o755)
	os.WriteFile(filepath.Join(root, "app-b", "go.mod"), []byte("module app-b\n"), 0o644)

	os.MkdirAll(filepath.Join(root, "plain-folder"), 0o755)
	os.MkdirAll(filepath.Join(root, "app-a", "node_modules", "foo"), 0o755)

	results, err := DetectProjects(root, 1)
	if err != nil {
		t.Fatalf("DetectProjects failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 directories, got %d", len(results))
	}

	names := []string{results[0].Name, results[1].Name}
	for _, result := range results {
		names = append(names, result.Name)
	}
	if !slices.Contains(names, "app-a") || !slices.Contains(names, "app-b") || !slices.Contains(names, "plain-folder") || !slices.Contains(names, "src") {
		t.Errorf("expected app-a, app-b, plain-folder, and src, got %v", names)
	}

	for _, r := range results {
		if r.Name == "app-a" {
			if !slices.Contains(r.DetectedStack, "node") {
				t.Errorf("expected app-a to have node stack, got %v", r.DetectedStack)
			}
		}
		if r.Name == "app-b" {
			if !slices.Contains(r.DetectedStack, "go") {
				t.Errorf("expected app-b to have go stack, got %v", r.DetectedStack)
			}
		}
		if r.Name == "plain-folder" && len(r.DetectedStack) != 0 {
			t.Errorf("expected plain-folder to have no detected stack, got %v", r.DetectedStack)
		}
	}
}

func TestDetectStack(t *testing.T) {
	tests := []struct {
		files    map[string]string
		expected []string
	}{
		{
			files: map[string]string{
				"package.json": `{"dependencies":{"next":"14"}}`,
			},
			expected: []string{"node", "nextjs"},
		},
		{
			files: map[string]string{
				"go.mod": "module example\n",
			},
			expected: []string{"go"},
		},
		{
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"example\"\n",
			},
			expected: []string{"rust"},
		},
	}

	for _, tt := range tests {
		dir := t.TempDir()
		for name, content := range tt.files {
			os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
		}
		stack := detectStack(dir)
		for _, exp := range tt.expected {
			if !slices.Contains(stack, exp) {
				t.Errorf("expected stack to contain %q, got %v", exp, stack)
			}
		}
	}
}
