package fsx

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/enes/project-brain-mcp/internal/app"
)

// Roots manages configured filesystem roots.
type Roots struct {
	mu    sync.RWMutex
	roots map[string]app.RootConfig
}

// NewRoots creates a Roots manager from config.
func NewRoots(cfg []app.RootConfig) *Roots {
	r := &Roots{roots: make(map[string]app.RootConfig, len(cfg))}
	for _, rc := range cfg {
		r.roots[rc.ID] = rc
	}
	return r
}

// Get returns a root by ID.
func (r *Roots) Get(id string) (app.RootConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rc, ok := r.roots[id]
	return rc, ok
}

// All returns all roots.
func (r *Roots) All() []app.RootConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]app.RootConfig, 0, len(r.roots))
	for _, rc := range r.roots {
		out = append(out, rc)
	}
	return out
}

// ResolveProject resolves a project_id to an absolute filesystem path.
// project_id format: "root_id:project_name" where project_name is a relative path.
func (r *Roots) ResolveProject(projectID string) (app.RootConfig, string, error) {
	parts := strings.SplitN(projectID, ":", 2)
	if len(parts) != 2 {
		return app.RootConfig{}, "", fmt.Errorf("invalid project_id format")
	}
	rootID, relPath := parts[0], parts[1]
	root, ok := r.Get(rootID)
	if !ok {
		return app.RootConfig{}, "", fmt.Errorf("unknown root: %s", rootID)
	}
	absPath, err := secureJoin(root.Path, relPath)
	if err != nil {
		return app.RootConfig{}, "", fmt.Errorf("resolve project path: %w", err)
	}
	return root, absPath, nil
}

// secureJoin joins base and rel safely, preventing traversal attacks.
func secureJoin(base, rel string) (string, error) {
	cleanRel := filepath.Clean(rel)
	if strings.HasPrefix(cleanRel, "..") || strings.HasPrefix(cleanRel, "/") || strings.HasPrefix(cleanRel, `\`) {
		return "", fmt.Errorf("path traversal detected: %s", rel)
	}
	joined := filepath.Join(base, cleanRel)
	resolvedBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		resolvedBase = base
	}
	resolvedJoined, err := filepath.EvalSymlinks(joined)
	if err != nil {
		// Path may not exist yet; resolve the parent instead.
		resolvedJoined = joined
	}
	if !strings.HasPrefix(filepath.Clean(resolvedJoined), filepath.Clean(resolvedBase)+string(filepath.Separator)) &&
		filepath.Clean(resolvedJoined) != filepath.Clean(resolvedBase) {
		return "", fmt.Errorf("path escapes root: %s", joined)
	}
	return joined, nil
}
