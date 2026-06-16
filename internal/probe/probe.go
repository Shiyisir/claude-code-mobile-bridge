package probe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/term"

	"github.com/shiyisir/claude-proxy/internal/config"
	"github.com/shiyisir/claude-proxy/internal/logging"
	"github.com/shiyisir/claude-proxy/internal/security"
)

// Record contains everything the probe captures about one invocation.
type Record struct {
	Timestamp string            `json:"timestamp"`
	Proxy     ProxyInfo         `json:"proxy"`
	Call      CallInfo          `json:"call"`
	Config    ConfigInfo        `json:"config"`
	Terminal  TerminalInfo      `json:"terminal"`
	Sample    *SampleInfo       `json:"sample,omitempty"`
}

// ProxyInfo identifies the proxy process itself.
type ProxyInfo struct {
	ExePath string `json:"exe_path"`
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
}

// CallInfo captures how the proxy was invoked.
type CallInfo struct {
	Argv []string          `json:"argv"`
	CWD  string            `json:"cwd"`
	Env  map[string]string `json:"env"`
}

// ConfigInfo records the resolved real_bin status.
type ConfigInfo struct {
	RealBin     string `json:"real_bin"`
	RealBinOK   bool   `json:"real_bin_ok"`
	RealBinIsExe bool  `json:"real_bin_is_exe"`
	SelfPath    string `json:"self_path"`
	NotRecursive bool  `json:"not_recursive"`
}

// TerminalInfo records whether each stream is a terminal.
type TerminalInfo struct {
	StdinIsTerminal  bool `json:"stdin_is_terminal"`
	StdoutIsTerminal bool `json:"stdout_is_terminal"`
	StderrIsTerminal bool `json:"stderr_is_terminal"`
}

// SampleInfo records truncated output samples.
type SampleInfo struct {
	StdoutSample string `json:"stdout_sample,omitempty"`
	StderrSample string `json:"stderr_sample,omitempty"`
	MaxBytes     int    `json:"max_bytes"`
}

const maxSampleBytes = 64 * 1024 // 64KB

// Capture creates a probe record from the current invocation context.
func Capture(cfg *config.Config, args []string) *Record {
	exe, _ := os.Executable()
	now := time.Now().Format("20060102-150405")

	rec := &Record{
		Timestamp: now,
		Proxy: ProxyInfo{
			ExePath: exe,
			Version: "0.1.0",
			GOOS:    runtime.GOOS,
			GOARCH:  runtime.GOARCH,
			PID:     os.Getpid(),
			PPID:    os.Getppid(),
		},
		Call: CallInfo{
			Argv: args,
			CWD:  mustGetwd(),
			Env:  security.FilterEnv(),
		},
		Config: ConfigInfo{
			RealBin:      cfg.RealBin,
			RealBinOK:    fileExists(cfg.RealBin),
			SelfPath:     exe,
			NotRecursive: !sameFile(cfg.RealBin, exe),
		},
		Terminal: TerminalInfo{
			StdinIsTerminal:  term.IsTerminal(int(os.Stdin.Fd())),
			StdoutIsTerminal: term.IsTerminal(int(os.Stdout.Fd())),
			StderrIsTerminal: term.IsTerminal(int(os.Stderr.Fd())),
		},
	}

	return rec
}

// Write saves the probe record to the probe log directory.
func Write(rec *Record) error {
	dir := probeLogDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("probe: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, fmt.Sprintf("probe-%s.json", rec.Timestamp))

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("probe: marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("probe: write %s: %w", path, err)
	}
	logging.Info("probe record written", "path", path)
	return nil
}

// AttachSample adds stdout/stderr samples to the record and re-saves.
// All samples are redacted to prevent credential leaks.
func AttachSample(rec *Record, stdoutSample, stderrSample []byte) {
	rec.Sample = &SampleInfo{
		MaxBytes: maxSampleBytes,
	}
	stdout := string(stdoutSample)
	stderr := string(stderrSample)
	if len(stdoutSample) > maxSampleBytes {
		stdout = string(stdoutSample[:maxSampleBytes])
	}
	if len(stderrSample) > maxSampleBytes {
		stderr = string(stderrSample[:maxSampleBytes])
	}
	rec.Sample.StdoutSample = security.RedactString(stdout)
	rec.Sample.StderrSample = security.RedactString(stderr)
	_ = Write(rec) // best-effort update with samples
}

func probeLogDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cc-connect", "claude-proxy", "logs")
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
