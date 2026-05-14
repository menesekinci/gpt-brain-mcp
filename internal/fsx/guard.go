package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enes/project-brain-mcp/internal/app"
)

// Guard enforces filesystem security policies.
type Guard struct {
	ignoreDirs     map[string]struct{}
	ignoreFiles    map[string]struct{}
	secretFiles    []string
	secretPatterns []string
}

// NewGuard creates a Guard from configuration.
func NewGuard(cfg app.IgnoreConfig) *Guard {
	g := &Guard{
		ignoreDirs:  make(map[string]struct{}, len(cfg.Dirs)),
		ignoreFiles: make(map[string]struct{}, len(cfg.Files)),
		secretFiles: []string{
			".env", ".envrc", ".env.local", ".env.production", ".env.development",
			"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa",
			".gitconfig", ".git-credentials",
		},
		secretPatterns: []string{
			"*.pem", "*.key", "*.pfx", "*.p12", "*.crt",
			"secrets.*", "credentials.*", "*secret*", "*credential*",
		},
	}
	for _, d := range cfg.Dirs {
		g.ignoreDirs[d] = struct{}{}
	}
	for _, f := range cfg.Files {
		g.ignoreFiles[f] = struct{}{}
	}
	return g
}

// IsPathAllowed checks if a relative path is allowed for reading/writing.
func (g *Guard) IsPathAllowed(relPath string, mode string, root app.RootConfig) error {
	clean := filepath.Clean(relPath)
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("path traversal")
		}
		if g.isIgnoredDir(part) {
			return fmt.Errorf("ignored directory: %s", part)
		}
		if g.isSecretFile(part) {
			return fmt.Errorf("sensitive file blocked")
		}
	}
	return nil
}

// IsWriteAllowed checks if writing to a path is allowed.
func (g *Guard) IsWriteAllowed(absPath string, projectAbsPath string, root app.RootConfig) error {
	if root.ReadOnly {
		return fmt.Errorf("root is read-only")
	}
	// Ensure path is within project.
	if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(projectAbsPath)+string(filepath.Separator)) {
		return fmt.Errorf("path escapes project")
	}
	rel, err := filepath.Rel(projectAbsPath, absPath)
	if err != nil {
		return fmt.Errorf("cannot relativize path")
	}
	if strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path escapes project")
	}
	parts := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	for _, part := range parts {
		if g.isSecretFile(part) {
			return fmt.Errorf("sensitive file blocked")
		}
	}
	if filepath.Clean(rel) == "AGENTS.md" {
		return nil
	}
	// Must be under one of the writable plan directories.
	if len(root.WritablePlanDirs) > 0 {
		allowed := false
		for _, wd := range root.WritablePlanDirs {
			if len(parts) > 0 && parts[0] == wd {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("writes only allowed under %v", root.WritablePlanDirs)
		}
	}
	return nil
}

func (g *Guard) isIgnoredDir(name string) bool {
	_, ok := g.ignoreDirs[name]
	return ok
}

func (g *Guard) isSecretFile(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range g.secretFiles {
		if lower == s {
			return true
		}
	}
	for _, p := range g.secretPatterns {
		matched, _ := filepath.Match(p, lower)
		if matched {
			return true
		}
		matched, _ = filepath.Match(p, name)
		if matched {
			return true
		}
	}
	return false
}

// ShouldSkipDir returns true if a directory should be skipped during traversal.
func (g *Guard) ShouldSkipDir(name string) bool {
	return g.isIgnoredDir(name) || name == ".git"
}

// IsSensitivePath checks if a path is sensitive regardless of context.
func (g *Guard) IsSensitivePath(path string) bool {
	base := filepath.Base(path)
	if g.isSecretFile(base) {
		return true
	}
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for _, p := range parts {
		if strings.EqualFold(p, ".ssh") || strings.EqualFold(p, ".aws") ||
			strings.EqualFold(p, ".gcp") || strings.EqualFold(p, ".azure") ||
			strings.EqualFold(p, ".kube") {
			return true
		}
	}
	return false
}

// IsBinaryFile detects if a file is likely binary by reading a sample.
func IsBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil {
		return false, err
	}
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true, nil
		}
	}
	return false, nil
}
