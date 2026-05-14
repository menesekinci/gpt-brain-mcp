package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RootConfig defines an allowed filesystem root.
type RootConfig struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Path            string   `yaml:"path"`
	WritablePlanDirs []string `yaml:"writable_plan_dirs"`
	ReadOnly        bool     `yaml:"read_only"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	Mode           string `yaml:"mode"`
	OwnerEmail     string `yaml:"owner_email"`
	MaxFileBytes   int    `yaml:"max_file_bytes"`
	MaxTreeEntries int    `yaml:"max_tree_entries"`
	MaxSearchResults int  `yaml:"max_search_results"`
	AllowBinaryFiles bool `yaml:"allow_binary_files"`
	RedactSecrets  bool   `yaml:"redact_secrets"`
	AuditLog       bool   `yaml:"audit_log"`
}

// ServerConfig holds server settings.
type ServerConfig struct {
	ListenAddr     string `yaml:"listen_addr"`
	PublicBaseURL  string `yaml:"public_base_url"`
	MCPPath        string `yaml:"mcp_path"`
}

// IgnoreConfig defines ignore patterns.
type IgnoreConfig struct {
	Dirs  []string `yaml:"dirs"`
	Files []string `yaml:"files"`
}

// AuthConfig defines authentication settings.
type AuthConfig struct {
	Type              string `yaml:"type"`
	IssuerURL         string `yaml:"issuer_url"`
	ClientID          string `yaml:"client_id"`
	RedirectURI       string `yaml:"redirect_uri"`
	OwnerSecret       string `yaml:"owner_secret"`
	JWTSecret         string `yaml:"jwt_secret"`
	TokenExpiryMinutes int   `yaml:"token_expiry_minutes"`
}

// Config is the top-level configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	Roots    []RootConfig   `yaml:"roots"`
	Ignore   IgnoreConfig   `yaml:"ignore"`
	Auth     AuthConfig     `yaml:"auth"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Server: ServerConfig{
			ListenAddr:    "127.0.0.1:3939",
			PublicBaseURL: "",
			MCPPath:       "/mcp",
		},
		Security: SecurityConfig{
			Mode:             "planning_write",
			MaxFileBytes:     250000,
			MaxTreeEntries:   3000,
			MaxSearchResults: 100,
			AllowBinaryFiles: false,
			RedactSecrets:    true,
			AuditLog:         true,
		},
		Roots: []RootConfig{
			{
				ID:               "default",
				Name:             "Default Projects",
				Path:             filepath.Join(home, "Projects"),
				WritablePlanDirs: []string{".chatgpt", ".ai"},
				ReadOnly:         false,
			},
		},
		Ignore: IgnoreConfig{
			Dirs: []string{
				".git", "node_modules", "vendor", "dist", "build",
				".next", ".turbo", ".cache", ".venv", "venv",
				"__pycache__", ".pytest_cache", ".mypy_cache",
				"target", "bin", "obj", ".idea", ".vscode",
			},
			Files: []string{
				".env", ".env.*", ".envrc",
				"id_rsa", "id_ed25519", "id_ecdsa",
				"*.pem", "*.key", "*.pfx", "*.p12",
				"secrets.*", "credentials.*",
				".DS_Store", "Thumbs.db",
			},
		},
		Auth: AuthConfig{
			Type: "noauth_dev",
		},
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	for i := range cfg.Roots {
		if cfg.Roots[i].Path == "" {
			continue
		}
		expanded, err := expandPath(cfg.Roots[i].Path)
		if err != nil {
			return nil, fmt.Errorf("expand root path %q: %w", cfg.Roots[i].Path, err)
		}
		cfg.Roots[i].Path = expanded
	}
	return cfg, nil
}

func expandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}
