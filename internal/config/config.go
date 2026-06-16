package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultPath is the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cc-connect", "claude-proxy", "config.json")
}

// DefaultTokenPath returns the default ws-token file path.
func DefaultTokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cc-connect", "claude-proxy", "runtime", "ws-token")
}

// Config holds all claude-proxy configuration.
type Config struct {
	RealBin       string `json:"real_bin"`          // absolute path to real Claude binary
	WSHost        string `json:"ws_host"`           // default "127.0.0.1"
	WSPort        int    `json:"ws_port"`           // default 9876
	WSTokenFile   string `json:"ws_token_file"`     // default runtime/ws-token under cc-connect
	MaxWSClients  int    `json:"max_ws_clients"`    // default 4
	MaxWSQueue    int    `json:"max_ws_queue"`      // default 256 per client
	LogLevel      string `json:"log_level"`         // debug, info, warn, error
	EnableJSON    bool   `json:"enable_json_parse"` // default true
	EnableWS      bool   `json:"enable_ws"`         // default true
}

// Load reads config from the default path.
func Load() (*Config, error) {
	path := DefaultPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})

	cfg := &Config{
		WSHost:       "127.0.0.1",
		WSPort:       9876,
		WSTokenFile:  DefaultTokenPath(),
		MaxWSClients: 4,
		MaxWSQueue:   256,
		LogLevel:     "info",
		EnableJSON:   true,
		EnableWS:     false, // disabled by default; use JSONL polling instead
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Validate checks config for safety and correctness.
func Validate(cfg *Config) error {
	if cfg.RealBin == "" {
		return fmt.Errorf("real_bin is required")
	}
	if !filepath.IsAbs(cfg.RealBin) {
		return fmt.Errorf("real_bin must be absolute path, got %s", cfg.RealBin)
	}
	self, _ := os.Executable()
	if self != "" && sameFile(cfg.RealBin, self) {
		return fmt.Errorf("real_bin must not be the proxy executable itself: %s", cfg.RealBin)
	}
	info, err := os.Stat(cfg.RealBin)
	if err != nil {
		return fmt.Errorf("real_bin not accessible: %s: %w", cfg.RealBin, err)
	}
	if info.IsDir() {
		return fmt.Errorf("real_bin is a directory, not an executable: %s", cfg.RealBin)
	}
	if cfg.EnableWS {
		if cfg.WSHost != "127.0.0.1" && cfg.WSHost != "localhost" {
			return fmt.Errorf("ws_host must be 127.0.0.1, got %s", cfg.WSHost)
		}
		if cfg.MaxWSClients < 1 {
			cfg.MaxWSClients = 4
		}
		if cfg.MaxWSQueue < 1 {
			cfg.MaxWSQueue = 256
		}
	}
	return nil
}

// LookupRealBin scans PATH for all "claude*" entries.
func LookupRealBin() []string {
	var found []string
	for _, name := range []string{"claude", "claude.exe", "claude.cmd", "claude.ps1"} {
		if p, err := exec.LookPath(name); err == nil {
			found = append(found, p)
		}
	}
	return found
}

func sameFile(a, b string) bool {
	infoA, err := os.Stat(a)
	if err != nil {
		return false
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}
