//go:build windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/shiyisir/claude-proxy/internal/config"
	"github.com/shiyisir/claude-proxy/internal/logging"
)

// Fallback runs the real Claude as a subprocess and forwards its exit code.
// On Windows, syscall.Exec is not reliable — use exec.Command instead.
func Fallback(args []string) {
	cfg, err := config.Load()
	if err != nil {
		logging.Error("fallback: cannot load config", "error", err)
		// Last resort: try "claude" from PATH (may recurse, but better than nothing)
		binary, _ := exec.LookPath("claude")
		if binary == "" {
			logging.Error("fallback: claude not found in PATH, giving up")
			os.Exit(1)
		}
		runAndExit(binary, args)
		return
	}

	if _, err := os.Stat(cfg.RealBin); err != nil {
		logging.Error("fallback: real_bin not accessible",
			"real_bin", cfg.RealBin, "error", err)
		fmt.Fprintf(os.Stderr, "claude-proxy: real_bin not found: %s\n", cfg.RealBin)
		os.Exit(1)
	}

	runAndExit(cfg.RealBin, args)
}

// runAndExit executes a command with full stdio forwarding and exits with its exit code.
func runAndExit(binary string, args []string) {
	cmd := exec.Command(binary, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "claude-proxy: %s: %v\n", binary, err)
		os.Exit(1)
	}
	os.Exit(0)
}
