package project

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectSignals are files that indicate a project root.
var ProjectSignals = []string{
	"package.json",
	"pnpm-lock.yaml", "yarn.lock", "package-lock.json", "bun.lockb",
	"go.mod",
	"Cargo.toml",
	"pyproject.toml", "requirements.txt", "setup.py", "Pipfile",
	"composer.json",
	"deno.json", "deno.jsonc",
	"Makefile",
	"Dockerfile",
	"Dockerfile.dev",
	".git",
	"tsconfig.json",
	"vite.config.ts", "vite.config.js",
	"next.config.js", "next.config.ts", "next.config.mjs",
	"angular.json",
	"pom.xml", "build.gradle",
	"CMakeLists.txt",
	"pubspec.yaml",
}

// DetectResult holds project detection info.
type DetectResult struct {
	Name          string   `json:"name"`
	RelativePath  string   `json:"relative_path"`
	DetectedStack []string `json:"detected_stack"`
	HasGit        bool     `json:"has_git"`
}

// DetectProjects lists directories under a root and annotates those with project signals.
func DetectProjects(rootPath string, maxDepth int) ([]DetectResult, error) {
	var results []DetectResult
	seen := make(map[string]struct{})

	var walk func(string, int) error
	walk = func(current string, depth int) error {
		if depth > maxDepth {
			return nil
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" {
				continue
			}
			subPath := filepath.Join(current, name)
			if _, ok := seen[subPath]; ok {
				continue
			}
			rel, _ := filepath.Rel(rootPath, subPath)
			rel = filepath.ToSlash(rel)
			seen[subPath] = struct{}{}
			stack := detectStack(subPath)
			_, hasGit := os.Stat(filepath.Join(subPath, ".git"))
			results = append(results, DetectResult{
				Name:          name,
				RelativePath:  rel,
				DetectedStack: stack,
				HasGit:        hasGit == nil,
			})
			_ = walk(subPath, depth+1)
		}
		return nil
	}

	_ = walk(rootPath, 0)
	// Also consider the root itself when it has project signals.
	if hasProjectSignal(rootPath) {
		name := filepath.Base(rootPath)
		results = append([]DetectResult{{
			Name:          name,
			RelativePath:  ".",
			DetectedStack: detectStack(rootPath),
			HasGit:        false,
		}}, results...)
	}
	return results, nil
}

func hasProjectSignal(path string) bool {
	for _, signal := range ProjectSignals {
		if _, err := os.Stat(filepath.Join(path, signal)); err == nil {
			return true
		}
	}
	return false
}

func detectStack(path string) []string {
	var stack []string
	if exists(filepath.Join(path, "package.json")) {
		stack = append(stack, "node")
		data, _ := os.ReadFile(filepath.Join(path, "package.json"))
		content := string(data)
		if strings.Contains(content, `"next"`) {
			stack = append(stack, "nextjs")
		}
		if strings.Contains(content, `"react"`) {
			stack = append(stack, "react")
		}
		if strings.Contains(content, `"vue"`) {
			stack = append(stack, "vue")
		}
		if strings.Contains(content, `"svelte"`) {
			stack = append(stack, "svelte")
		}
		if strings.Contains(content, `"typescript"`) || strings.Contains(content, `"ts-node"`) {
			stack = append(stack, "typescript")
		}
		if strings.Contains(content, `"tailwindcss"`) || strings.Contains(content, `"tailwind"`) {
			stack = append(stack, "tailwind")
		}
		if strings.Contains(content, `"vite"`) {
			stack = append(stack, "vite")
		}
		if strings.Contains(content, `"express"`) {
			stack = append(stack, "express")
		}
		if strings.Contains(content, `"prisma"`) {
			stack = append(stack, "prisma")
		}
	}
	if exists(filepath.Join(path, "go.mod")) {
		stack = append(stack, "go")
	}
	if exists(filepath.Join(path, "Cargo.toml")) {
		stack = append(stack, "rust")
	}
	if exists(filepath.Join(path, "pyproject.toml")) || exists(filepath.Join(path, "requirements.txt")) {
		stack = append(stack, "python")
		if exists(filepath.Join(path, "pyproject.toml")) {
			data, _ := os.ReadFile(filepath.Join(path, "pyproject.toml"))
			content := string(data)
			if strings.Contains(content, `[tool.poetry]`) {
				stack = append(stack, "poetry")
			}
			if strings.Contains(content, `[project]`) {
				stack = append(stack, "pip")
			}
		}
	}
	if exists(filepath.Join(path, "composer.json")) {
		stack = append(stack, "php")
	}
	if exists(filepath.Join(path, "Dockerfile")) || exists(filepath.Join(path, "docker-compose.yml")) {
		stack = append(stack, "docker")
	}
	if exists(filepath.Join(path, "tsconfig.json")) {
		if !contains(stack, "typescript") {
			stack = append(stack, "typescript")
		}
	}
	return deduplicate(stack)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func deduplicate(slice []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
