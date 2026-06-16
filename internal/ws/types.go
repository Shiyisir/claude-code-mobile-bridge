package ws

import "encoding/json"

// Message types sent to WebSocket clients.
const (
	TypeEvent = "event"
	TypeError = "error"
	TypePing  = "ping"
)

// OutMsg is a message sent to a WebSocket client.
type OutMsg struct {
	Type      string          `json:"type"`
	Version   int             `json:"version"`
	SessionID string          `json:"session_id,omitempty"`
	Event     json.RawMessage `json:"event,omitempty"`
	Error     *ErrorInfo      `json:"error,omitempty"`
	TS        string          `json:"ts,omitempty"`
}

// ErrorInfo is included in error messages.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
