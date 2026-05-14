package fsx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enes/project-brain-mcp/internal/app"
)

// SearchResult represents a single match.
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// SearchProject searches for a query string within a project.
func SearchProject(projectID string, projectAbsPath string, root app.RootConfig, guard *Guard, query string, glob string, maxResults int) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	queryLower := strings.ToLower(query)
	var results []SearchResult
	err := filepath.WalkDir(projectAbsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if d.IsDir() {
			if guard.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(projectAbsPath, path)
		if guard.IsPathAllowed(rel, "read", root) != nil {
			return nil
		}
		if guard.IsSensitivePath(path) {
			return nil
		}
		if glob != "" {
			matched, _ := filepath.Match(glob, filepath.Base(path))
			if !matched {
				// Try path-based glob.
				matched, _ = filepath.Match(glob, rel)
			}
			if !matched {
				return nil
			}
		}
		isBin, _ := IsBinaryFile(path)
		if isBin {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), queryLower) {
				results = append(results, SearchResult{
					Path:    rel,
					Line:    lineNum,
					Content: strings.TrimSpace(line),
				})
				if len(results) >= maxResults {
					f.Close()
					return fmt.Errorf("max_results reached")
				}
			}
		}
		return nil
	})
	if err != nil && err.Error() == "max_results reached" {
		err = nil
	}
	return results, err
}
