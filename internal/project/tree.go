package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// TreeEntry represents a file or directory in the tree.
type TreeEntry struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size,omitempty"`
	Children int    `json:"children,omitempty"`
}

// GetProjectTree returns a filtered file tree for a project.
func GetProjectTree(absPath string, maxEntries int, depth int, ignoreDirs []string) ([]TreeEntry, error) {
	var results []TreeEntry
	ignore := make(map[string]struct{})
	for _, d := range ignoreDirs {
		ignore[d] = struct{}{}
	}

	var walk func(string, int) error
	walk = func(current string, level int) error {
		if level > depth {
			return nil
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if len(results) >= maxEntries {
				return fmt.Errorf("max_entries reached")
			}
			name := entry.Name()
			if name == ".git" {
				if level == 0 {
					continue
				}
			}
			if _, ok := ignore[name]; ok {
				if entry.IsDir() {
					continue
				}
				continue
			}
			rel, _ := filepath.Rel(absPath, filepath.Join(current, name))
			rel = filepath.ToSlash(rel)
			if entry.IsDir() {
				results = append(results, TreeEntry{Path: rel, IsDir: true})
				_ = walk(filepath.Join(current, name), level+1)
			} else {
				info, _ := entry.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				results = append(results, TreeEntry{Path: rel, IsDir: false, Size: size})
			}
		}
		return nil
	}

	err := walk(absPath, 0)
	if err != nil && err.Error() == "max_entries reached" {
		err = nil
	}
	return results, err
}
