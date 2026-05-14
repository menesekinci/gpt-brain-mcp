package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enes/project-brain-mcp/internal/app"
)

// WriteFile writes data to a path within a project, enforcing write guards.
func WriteFile(projectID, relPath string, data []byte, projectAbsPath string, root app.RootConfig, guard *Guard, maxBytes int) error {
	absPath := filepath.Join(projectAbsPath, relPath)
	if err := guard.IsWriteAllowed(absPath, projectAbsPath, root); err != nil {
		return fmt.Errorf("blocked: %w", err)
	}
	// Ensure resolved path is still within project.
	if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(projectAbsPath)+string(filepath.Separator)) {
		return fmt.Errorf("path escapes project")
	}
	if len(data) > maxBytes {
		return fmt.Errorf("content too large: %d bytes (max %d)", len(data), maxBytes)
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	// Atomic write: write to temp then rename.
	tmpPath := absPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalize write: %w", err)
	}
	return nil
}
