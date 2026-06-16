package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/shiyisir/claude-proxy/internal/config"
	"github.com/shiyisir/claude-proxy/internal/launcher"
	"github.com/shiyisir/claude-proxy/internal/logging"
)

func main() {
	logging.Init()

	// VS Code passes the original binary path as os.Args[1] when using
	// claudeCode.claudeProcessWrapper. Filter it out so the real Claude
	// only receives the actual arguments.
	realArgs := filterArgs(os.Args[1:])

	// Phase 0: main flow — load config, validate, capture probe, launch real Claude.
	// No WebSocket, no JSON parser, no stdin injection yet.

	cfg, err := config.Load()
	if err != nil {
		logging.Error("cannot load config, falling back", "error", err)
		launcher.Fallback(realArgs)
		os.Exit(1)
	}

	if err := config.Validate(cfg); err != nil {
		logging.Error("config validation failed", "error", err)
		fmt.Fprintf(os.Stderr, "claude-proxy: config error: %v\n", err)
		launcher.Fallback(realArgs)
		os.Exit(1)
	}

	logging.Debug("claude-proxy starting",
		"real_bin", cfg.RealBin,
		"pid", os.Getpid(),
		"ppid", os.Getppid(),
	)

	if err := launcher.Start(cfg, realArgs); err != nil {
		logging.Error("launcher failed", "error", err)
		fmt.Fprintf(os.Stderr, "claude-proxy: real claude failed: %v\n", err)
		os.Exit(1)
	}
}

// filterArgs removes the VS Code injected binary path from the argument list.
// When claudeCode.claudeProcessWrapper is set, VS Code passes the original
// binary path as the first argument. We need to skip it.
func filterArgs(args []string) []string {
	if len(args) > 0 && strings.HasSuffix(strings.ToLower(args[0]), ".exe") {
		return args[1:]
	}
	return args
}
