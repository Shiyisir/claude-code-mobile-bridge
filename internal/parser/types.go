package parser

import "encoding/json"

// EventKind classifies a normalized Claude Code event.
type EventKind string

const (
	KindSessionStart       EventKind = "session_start"
	KindUserMessage        EventKind = "user_message"
	KindAssistantMessage   EventKind = "assistant_message"
	KindAssistantDelta     EventKind = "assistant_delta"
	KindToolUse            EventKind = "tool_use"
	KindToolResult         EventKind = "tool_result"
	KindPermissionRequest  EventKind = "permission_request"
	KindDiff               EventKind = "diff"
	KindUsage              EventKind = "usage"
	KindError              EventKind = "error"
	KindUnknown            EventKind = "unknown"
)

// NormalizedEvent is the standardized output written to JSONL.
type NormalizedEvent struct {
	TS        string         `json:"ts"`
	SessionID string         `json:"session_id"`
	Source    string         `json:"source"` // "stdout" or "stderr"
	Kind      EventKind      `json:"kind"`
	RawType   string         `json:"raw_type"`
	Payload   map[string]any `json:"payload"`
	Raw       map[string]any `json:"raw,omitempty"`
}

// NewNormalizedEvent creates a NormalizedEvent with defaults.
func NewNormalizedEvent(source, rawType string, raw map[string]any) *NormalizedEvent {
	e := &NormalizedEvent{
		Source:  source,
		RawType: rawType,
		Raw:     raw,
	}
	// Extract session_id from raw event
	if sid, ok := raw["session_id"].(string); ok {
		e.SessionID = sid
	}
	return e
}

// RawFromLine attempts to parse a single NDJSON line into a map.
// Returns nil if the line is not valid JSON.
func RawFromLine(line []byte) map[string]any {
	if len(line) == 0 {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil
	}
	return raw
}
