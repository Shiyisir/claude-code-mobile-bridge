package parser

import (
	"encoding/json"
	"time"
)

// Normalize converts a raw NDJSON map to a NormalizedEvent.
func Normalize(cfg Config, source string, raw map[string]any) *NormalizedEvent {
	rawType, _ := raw["type"].(string)
	event := NewNormalizedEvent(source, rawType, raw)
	event.TS = time.Now().UTC().Format(time.RFC3339)

	if event.SessionID == "" {
		event.SessionID = extractSessionID(raw)
	}

	kind, payload := classify(rawType, raw)
	event.Kind = kind
	event.Payload = payload

	// Truncate raw to max bytes
	if cfg.MaxRawBytes > 0 {
		event.Raw = truncateMap(event.Raw, cfg.MaxRawBytes)
	}
	if event.Payload != nil {
		event.Payload = truncateMap(event.Payload, cfg.MaxRawBytes)
	}

	return event
}

func classify(rawType string, raw map[string]any) (EventKind, map[string]any) {
	switch rawType {
	case "system":
		return classifySystem(raw)
	case "user":
		return KindUserMessage, extractMessage(raw)
	case "assistant":
		return KindAssistantMessage, extractMessage(raw)
	case "stream_event":
		return classifyStreamEvent(raw)
	case "result":
		return classifyResult(raw)
	default:
		return KindUnknown, raw
	}
}

func classifySystem(raw map[string]any) (EventKind, map[string]any) {
	subtype, _ := raw["subtype"].(string)
	switch subtype {
	case "init", "startup":
		return KindSessionStart, nil
	case "permission_request", "permission":
		return KindPermissionRequest, raw
	default:
		return KindUnknown, raw
	}
}

func classifyStreamEvent(raw map[string]any) (EventKind, map[string]any) {
	event, _ := raw["event"].(map[string]any)
	if event == nil {
		return KindUnknown, raw
	}
	eventType, _ := event["type"].(string)
	switch eventType {
	case "content_block_start", "content_block_delta":
		return classifyContentBlock(event)
	case "message_delta":
		return KindAssistantDelta, event
	case "message_start":
		return KindAssistantMessage, event
	default:
		return KindUnknown, raw
	}
}

func classifyContentBlock(event map[string]any) (EventKind, map[string]any) {
	block, _ := event["content_block"].(map[string]any)
	if block == nil {
		return KindUnknown, event
	}
	blockType, _ := block["type"].(string)
	switch blockType {
	case "tool_use":
		return KindToolUse, map[string]any{
			"tool_name": block["name"],
			"tool_id":   block["id"],
			"input":     block["input"],
		}
	case "text":
		return KindAssistantDelta, map[string]any{"text": block["text"]}
	default:
		return KindUnknown, event
	}
}

func classifyResult(raw map[string]any) (EventKind, map[string]any) {
	subtype, _ := raw["subtype"].(string)
	if subtype == "error" {
		result, _ := raw["result"].(string)
		return KindError, map[string]any{"error": result}
	}
	if subtype == "success" {
		result, _ := raw["result"].(string)
		switch result {
		case "tool_use":
			return KindToolUse, raw
		case "tool_result":
			return KindToolResult, raw
		}
	}
	return KindUnknown, raw
}

func extractSessionID(raw map[string]any) string {
	for _, key := range []string{"session_id", "uuid"} {
		if s, ok := raw[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func extractMessage(raw map[string]any) map[string]any {
	p := make(map[string]any)
	if msg, ok := raw["message"].(map[string]any); ok {
		if role, ok := msg["role"].(string); ok {
			p["role"] = role
		}
		if content, ok := msg["content"]; ok {
			p["content"] = content
		}
	}
	return p
}

// truncateMap recursively truncates string values to maxBytes.
func truncateMap(m map[string]any, maxBytes int) map[string]any {
	raw, err := json.Marshal(m)
	if err != nil {
		return m
	}
	if len(raw) <= maxBytes {
		return m
	}
	// Truncate: re-marshal with per-value truncation
	out := make(map[string]any)
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if len(val) > maxBytes/2 {
				out[k] = val[:maxBytes/2] + "...[truncated]"
			} else {
				out[k] = val
			}
		case map[string]any:
			out[k] = truncateMap(val, maxBytes/2)
		default:
			out[k] = v
		}
	}
	out["_truncated"] = true
	return out
}
