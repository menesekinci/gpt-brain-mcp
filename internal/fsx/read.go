package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enes/project-brain-mcp/internal/app"
)

// ReadFile reads an allowed file from a project, applying security checks.
func ReadFile(projectID, relPath string, projectAbsPath string, root app.RootConfig, guard *Guard, maxBytes int) ([]byte, error) {
	if err := guard.IsPathAllowed(relPath, "read", root); err != nil {
		return nil, fmt.Errorf("blocked: %w", err)
	}
	absPath := filepath.Join(projectAbsPath, relPath)
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return nil, fmt.Errorf("file not found")
	}
	// Ensure resolved path is still within project.
	if !strings.HasPrefix(filepath.Clean(resolved), filepath.Clean(projectAbsPath)+string(filepath.Separator)) {
		return nil, fmt.Errorf("file escapes project")
	}
	if guard.IsSensitivePath(resolved) {
		return nil, fmt.Errorf("blocked: sensitive file")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("file not found")
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > int64(maxBytes) {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxBytes)
	}
	isBin, err := IsBinaryFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot inspect file: %w", err)
	}
	if isBin {
		return nil, fmt.Errorf("binary files not allowed")
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	return data, nil
}
