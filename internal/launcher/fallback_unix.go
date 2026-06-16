//go:build !windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/shiyisir/claude-proxy/internal/config"
	"github.com/shiyisir/claude-proxy/internal/logging"
)

// Fallback directly execs the real Claude via syscall.Exec.
// On Unix, this replaces the current process image entirely.
func Fallback(args []string) {
	cfg, err := config.Load()
	if err != nil {
		logging.Error("fallback: cannot load config", "error", err)
		binary, _ := exec.LookPath("claude")
		if binary == "" {
			logging.Error("fallback: claude not found in PATH, giving up")
			os.Exit(1)
		}
		syscall.Exec(binary, append([]string{binary}, args...), os.Environ())
		return
	}

	if _, err := os.Stat(cfg.RealBin); err != nil {
		logging.Error("fallback: real_bin not accessible",
			"real_bin", cfg.RealBin, "error", err)
		fmt.Fprintf(os.Stderr, "claude-proxy: real_bin not found: %s\n", cfg.RealBin)
		os.Exit(1)
	}

	syscall.Exec(cfg.RealBin, append([]string{cfg.RealBin}, args...), os.Environ())
}
