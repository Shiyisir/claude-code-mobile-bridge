package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	Project    string
	Session    string
	EventsDir  string
	PollMs     int
	MaxChars   int
	MaxRetries int
}

func loadConfig() *config {
	home, _ := os.UserHomeDir()
	return &config{
		Project:    or(os.Getenv("BRIDGE_PROJECT"), "documents"),
		Session:    os.Getenv("BRIDGE_SESSION"),
		EventsDir:  or(os.Getenv("BRIDGE_EVENTS_DIR"), filepath.Join(home, ".cc-connect", "claude-proxy", "events")),
		PollMs:     2000,
		MaxChars:   2000,
		MaxRetries: 3,
	}
}

func main() {
	cfg := loadConfig()
	if cfg.Session == "" {
		fmt.Fprintln(os.Stderr, "bridge: BRIDGE_SESSION env is required")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "bridge: polling %s every %dms session=%s\n",
		cfg.EventsDir, cfg.PollMs, cfg.Session[:20])

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	readPos := make(map[string]int64)
	sent := make(map[string]bool)

	// Start from end of existing files to avoid history replay
	files, _ := filepath.Glob(filepath.Join(cfg.EventsDir, "session-*.jsonl"))
	for _, p := range files {
		if fi, err := os.Stat(p); err == nil {
			readPos[p] = fi.Size()
		}
	}

	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "bridge: stopped")
			return
		default:
		}

		files, _ := filepath.Glob(filepath.Join(cfg.EventsDir, "session-*.jsonl"))
		for _, path := range files {
			pos := readPos[path]

			f, err := os.Open(path)
			if err != nil {
				continue
			}
			fi, err := f.Stat()
			if err != nil || fi.Size() <= pos {
				f.Close()
				continue
			}

			f.Seek(pos, 0)
			buf := make([]byte, fi.Size()-pos)
			n, _ := f.Read(buf)
			f.Close()

			for _, line := range strings.Split(string(buf[:n]), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var ev jsonEvent
				if json.Unmarshal([]byte(line), &ev) != nil {
					continue
				}
				text := formatMsg(ev, cfg.MaxChars)
				if text == "" {
					continue
				}
				// Dedup by session+kind+content prefix
				key := fmt.Sprintf("%s:%s:%s", ev.SessionID, ev.Kind, text[:min(80, len(text))])
				if sent[key] {
					continue
				}
				sent[key] = true

				fmt.Fprintf(os.Stderr, "bridge: [%s] %s\n", ev.Kind, text[:min(100, len(text))])
				sendWithRetry(cfg, text)
			}
			readPos[path] = fi.Size()
		}

		if len(sent) > 10000 {
			sent = make(map[string]bool)
		}

		time.Sleep(time.Duration(cfg.PollMs) * time.Millisecond)
	}
}

func sendWithRetry(cfg *config, text string) {
	for i := 0; i < cfg.MaxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, "cc-connect", "send", "-p", cfg.Project, "-s", cfg.Session, "--stdin")
		cmd.Stdin = strings.NewReader(text)
		err := cmd.Run()
		cancel()
		if err == nil {
			return
		}
		if i < cfg.MaxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	fmt.Fprintf(os.Stderr, "bridge: send failed after %d retries\n", cfg.MaxRetries)
}

type jsonEvent struct {
	SessionID string         `json:"session_id"`
	Kind      string         `json:"kind"`
	RawType   string         `json:"raw_type"`
	Payload   map[string]any `json:"payload"`
}

func formatMsg(ev jsonEvent, maxChars int) string {
	var text string
	switch ev.Kind {
	case "assistant_message":
		if t := extractContent(ev.Payload); t != "" {
			text = "[Claude] " + t
		}
	case "user_message":
		if t := extractContent(ev.Payload); t != "" {
			text = "[User] " + t
		}
	case "tool_use":
		// Brief tool hint only
		if n, ok := ev.Payload["tool_name"].(string); ok {
			text = "[Tool] " + n
		}
	// Hide these to avoid noise:
	// case "tool_result", "session_start", "assistant_delta", "error", default
	default:
		return ""
	}
	if len(text) > maxChars {
		text = text[:maxChars] + "..."
	}
	return text
}

func extractContent(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if c, ok := payload["content"]; ok {
		switch v := c.(type) {
		case string:
			return v
		case []any:
			var parts []string
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					if t, ok := m["text"].(string); ok {
						parts = append(parts, t)
					}
				}
			}
			return strings.Join(parts, " ")
		}
	}
	if t, ok := payload["text"].(string); ok {
		return t
	}
	return ""
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
