package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InspectResult holds the full project inspection output.
type InspectResult struct {
	Name          string            `json:"name"`
	Stack         []string          `json:"stack"`
	Entrypoints   []string          `json:"entrypoints"`
	PackageManagers []string        `json:"package_managers"`
	ImportantFiles []string         `json:"important_files"`
	Summary       string            `json:"summary"`
	TreePreview   []string          `json:"tree_preview"`
	Manifests     map[string]any    `json:"manifests"`
	Warnings      []string          `json:"warnings"`
}

// InspectProject analyzes a project directory.
func InspectProject(absPath string, maxEntries int) (*InspectResult, error) {
	name := filepath.Base(absPath)
	stack := detectStack(absPath)
	entrypoints := findEntrypoints(absPath, stack)
	pms := findPackageManagers(absPath)
	important := findImportantFiles(absPath)
	manifests, _ := loadManifests(absPath)
	treeEntries, _ := GetProjectTree(absPath, maxEntries, 2, nil)
	var tree []string
	for _, e := range treeEntries {
		tree = append(tree, e.Path)
	}
	warnings := generateWarnings(absPath, stack, manifests)

	summary := fmt.Sprintf("Project %q uses %s.", name, strings.Join(stack, ", "))
	if len(stack) == 0 {
		summary = fmt.Sprintf("Project %q has no detectable stack.", name)
	}

	return &InspectResult{
		Name:           name,
		Stack:          stack,
		Entrypoints:    entrypoints,
		PackageManagers: pms,
		ImportantFiles: important,
		Summary:        summary,
		TreePreview:    tree,
		Manifests:      manifests,
		Warnings:       warnings,
	}, nil
}

func findEntrypoints(absPath string, stack []string) []string {
	var eps []string
	candidates := []string{
		"app/page.tsx", "app/page.ts", "app/page.jsx", "app/page.js",
		"src/main.ts", "src/main.tsx", "src/main.js", "src/main.jsx",
		"src/index.ts", "src/index.tsx", "src/index.js", "src/index.jsx",
		"main.go", "cmd/main.go", "cmd/server/main.go",
		"server.ts", "server.js",
		"app.py", "main.py", "manage.py",
		"src/lib.rs", "src/main.rs",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(absPath, c)); err == nil {
			eps = append(eps, c)
		}
	}
	return eps
}

func findPackageManagers(absPath string) []string {
	var pms []string
	if exists(filepath.Join(absPath, "pnpm-lock.yaml")) {
		pms = append(pms, "pnpm")
	} else if exists(filepath.Join(absPath, "yarn.lock")) {
		pms = append(pms, "yarn")
	} else if exists(filepath.Join(absPath, "package-lock.json")) {
		pms = append(pms, "npm")
	} else if exists(filepath.Join(absPath, "bun.lockb")) {
		pms = append(pms, "bun")
	}
	if exists(filepath.Join(absPath, "go.mod")) {
		pms = append(pms, "go modules")
	}
	if exists(filepath.Join(absPath, "Cargo.toml")) {
		pms = append(pms, "cargo")
	}
	if exists(filepath.Join(absPath, "pyproject.toml")) {
		pms = append(pms, "uv/poetry/hatch")
	} else if exists(filepath.Join(absPath, "requirements.txt")) {
		pms = append(pms, "pip")
	}
	return pms
}

func findImportantFiles(absPath string) []string {
	var files []string
	candidates := []string{
		"package.json", "tsconfig.json", "next.config.ts", "next.config.js",
		"go.mod", "go.sum",
		"Cargo.toml", "Cargo.lock",
		"pyproject.toml", "requirements.txt",
		"composer.json", "composer.lock",
		"README.md", "README",
		"Dockerfile", "docker-compose.yml",
		".gitignore",
		"Makefile",
		"vite.config.ts", "vite.config.js",
		"tailwind.config.ts", "tailwind.config.js",
	}
	for _, c := range candidates {
		if exists(filepath.Join(absPath, c)) {
			files = append(files, c)
		}
	}
	return files
}

func loadManifests(absPath string) (map[string]any, error) {
	manifests := make(map[string]any)
	if data, err := os.ReadFile(filepath.Join(absPath, "package.json")); err == nil {
		var m map[string]any
		if json.Unmarshal(data, &m) == nil {
			filtered := make(map[string]any)
			for _, k := range []string{"name", "version", "type", "scripts", "dependencies", "devDependencies"} {
				if v, ok := m[k]; ok {
					filtered[k] = v
				}
			}
			manifests["package.json"] = filtered
		}
	}
	if data, err := os.ReadFile(filepath.Join(absPath, "go.mod")); err == nil {
		lines := strings.Split(string(data), "\n")
		var trimmed []string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "//") {
				trimmed = append(trimmed, l)
			}
		}
		manifests["go.mod"] = strings.Join(trimmed, "\n")
	}
	if data, err := os.ReadFile(filepath.Join(absPath, "Cargo.toml")); err == nil {
		lines := strings.Split(string(data), "\n")
		var trimmed []string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "#") {
				trimmed = append(trimmed, l)
			}
		}
		manifests["Cargo.toml"] = strings.Join(trimmed, "\n")
	}
	return manifests, nil
}

func generateWarnings(absPath string, stack []string, manifests map[string]any) []string {
	var warnings []string
	if _, err := os.Stat(filepath.Join(absPath, ".env")); err == nil {
		warnings = append(warnings, ".env file present in project root (do not commit secrets)")
	}
	if len(stack) == 0 {
		warnings = append(warnings, "No detectable framework or language stack")
	}
	return warnings
}
