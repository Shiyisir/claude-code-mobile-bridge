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

func main() {
	ccProject := "documents"
	ccSession := os.Getenv("BRIDGE_SESSION")
	eventsDir := eventsDirPath()
	pollInterval := 2 * time.Second
	maxChars := 2000

	if ccSession == "" {
		fmt.Fprintln(os.Stderr, "bridge: BRIDGE_SESSION env is required")
		os.Exit(1)
	}
	if v := os.Getenv("BRIDGE_EVENTS_DIR"); v != "" {
		eventsDir = v
	}
	if v := os.Getenv("BRIDGE_PROJECT"); v != "" {
		ccProject = v
	}

	fmt.Fprintf(os.Stderr, "bridge: polling %s every %v, project=%s\n", eventsDir, pollInterval, ccProject)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// Start from current end of files — don't replay history
	readPositions := make(map[string]int64)
	sentSet := make(map[string]bool)

	// Initialize positions at end of existing files
	files, _ := filepath.Glob(filepath.Join(eventsDir, "session-*.jsonl"))
	for _, path := range files {
		fi, err := os.Stat(path)
		if err == nil {
			readPositions[path] = fi.Size()
		}
	}

	for {
		select {
		case <-sigCh:
			return
		default:
		}

		files, err := filepath.Glob(filepath.Join(eventsDir, "session-*.jsonl"))
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		for _, path := range files {
			pos := readPositions[path]

			f, err := os.Open(path)
			if err != nil {
				continue
			}
			fi, err := f.Stat()
			if err != nil {
				f.Close()
				continue
			}
			if fi.Size() <= pos {
				f.Close()
				continue
			}
			f.Seek(pos, 0)

			// Read new lines
			buf := make([]byte, fi.Size()-pos)
			n, _ := f.Read(buf)
			f.Close()

			lines := strings.Split(string(buf[:n]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Parse event
				var ev jsonEvent
				if err := json.Unmarshal([]byte(line), &ev); err != nil {
					continue
				}

				text := formatEventText(ev, maxChars)
				if text == "" {
					continue
				}

				// Dedup
				key := fmt.Sprintf("%s:%s:%s", ev.SessionID, ev.Kind, text[:min(80, len(text))])
				if sentSet[key] {
					continue
				}
				sentSet[key] = true

				fmt.Fprintf(os.Stderr, "bridge: sending [%s] %s\n", ev.Kind, text[:min(80, len(text))])
				go sendToCC(ccProject, ccSession, text)
			}

			readPositions[path] = fi.Size()
		}

		// Cleanup old dedup entries
		if len(sentSet) > 10000 {
			sentSet = make(map[string]bool)
		}

		time.Sleep(pollInterval)
	}
}

type jsonEvent struct {
	SessionID string `json:"session_id"`
	Kind      string `json:"kind"`
	RawType   string `json:"raw_type"`
	Payload   map[string]any `json:"payload"`
}

func formatEventText(ev jsonEvent, maxChars int) string {
	var text string
	switch ev.Kind {
	case "assistant_message":
		t := extractText(ev.Payload)
		if t != "" {
			text = "[Claude] " + t
		}
	case "user_message":
		t := extractText(ev.Payload)
		if t != "" {
			text = "[User] " + t
		}
	case "tool_use":
		name := ""
		if n, ok := ev.Payload["tool_name"].(string); ok {
			name = n
		}
		text = fmt.Sprintf("[Tool] %s", name)
	case "tool_result":
		text = "[Tool] 完成"
	case "session_start":
		text = "[System] 会话开始"
	default:
		return ""
	}
	if len(text) > maxChars {
		text = text[:maxChars] + "..."
	}
	return text
}

func extractText(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if content, ok := payload["content"]; ok {
		switch v := content.(type) {
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

func sendToCC(project, session, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cc-connect", "send", "-p", project, "-s", session, "--stdin")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "bridge: send failed: %v\n", err)
	}
}

func eventsDirPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cc-connect", "claude-proxy", "events")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
