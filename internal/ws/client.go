package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/shiyisir/claude-proxy/internal/eventbus"
	"github.com/shiyisir/claude-proxy/internal/logging"

	"nhooyr.io/websocket"
)

// Client represents a connected WebSocket client.
type Client struct {
	conn    *websocket.Conn
	sub     *subscriber
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	sessionID string // empty = all sessions
}

// handleClient manages a single WebSocket connection.
func (s *Server) handleClient(conn *websocket.Conn, sessionID string) {
	ctx, cancel := context.WithCancel(s.ctx)
	c := &Client{
		conn:      conn,
		ctx:       ctx,
		cancel:    cancel,
		sessionID: sessionID,
		sub:       &subscriber{ch: make(chan *eventbus.Event, s.cfg.MaxWSQueue)},
	}

	// Register with the active client list
	s.mu.Lock()
	s.clients[c] = struct{}{}
	s.mu.Unlock()

	logging.Info("ws: client connected", "session", sessionID)

	// Start writer goroutine
	c.wg.Add(1)
	go c.writeLoop()

	// Start reader goroutine (handles pings, close)
	c.wg.Add(1)
	go c.readLoop()

	c.wg.Wait()

	// Cleanup
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	c.conn.Close(websocket.StatusNormalClosure, "done")
	logging.Info("ws: client disconnected")
}

// sendEvent tries to send an event to the client. Non-blocking.
func (c *Client) sendEvent(ev *eventbus.Event) bool {
	if c.sessionID != "" && c.sessionID != ev.SessionID {
		return true // skip events for other sessions
	}
	select {
	case c.sub.ch <- ev:
		return true
	default:
		return false
	}
}

// writeLoop reads from the event channel and writes to the WebSocket.
func (c *Client) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.ctx.Done():
			return
		case ev, ok := <-c.sub.ch:
			if !ok {
				return
			}
			msg := OutMsg{
				Type:      TypeEvent,
				Version:   1,
				SessionID: ev.SessionID,
				Event:     ev.Data,
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			writeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			err = c.conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				logging.Warn("ws: write failed, closing client", "error", err)
				c.cancel()
				return
			}
		}
	}
}

// readLoop handles incoming messages (pings, close).
func (c *Client) readLoop() {
	defer c.wg.Done()
	defer c.cancel()
	for {
		_, _, err := c.conn.Read(c.ctx)
		if err != nil {
			return // connection closed
		}
		// In Phase 2, we don't process incoming messages beyond keeping the connection alive
	}
}

// subscriber is a per-client event queue.
type subscriber struct {
	ch chan *eventbus.Event
}

func (s *subscriber) Send(ev *eventbus.Event) bool {
	select {
	case s.ch <- ev:
		return true
	default:
		return false
	}
}
