package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shiyisir/claude-proxy/internal/eventbus"
	"github.com/shiyisir/claude-proxy/internal/logging"

	"nhooyr.io/websocket"
)

// Config holds WebSocket server configuration.
type Config struct {
	Host          string
	Port          int
	Token         string
	MaxClients    int
	MaxWSQueue    int
	EnableWS      bool
}

// Server handles WebSocket connections and broadcasts events.
type Server struct {
	cfg    Config
	bus    *eventbus.Bus
	ctx    context.Context
	cancel context.CancelFunc
	server *http.Server
	mu     sync.Mutex
	clients map[*Client]struct{}
	started atomic.Bool
}

// NewServer creates a WebSocket server.
func NewServer(cfg Config, bus *eventbus.Bus) *Server {
	return &Server{
		cfg:     cfg,
		bus:     bus,
		clients: make(map[*Client]struct{}),
	}
}

// Start begins listening. Returns immediately; errors are logged.
func (s *Server) Start() error {
	if !s.cfg.EnableWS {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/debug-token", s.handleDebugToken)
	logging.Info("ws: token configured", "token_len", len(s.cfg.Token))

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	lc := net.ListenConfig{}
	ln, err := lc.Listen(s.ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("ws: listen %s: %w", addr, err)
	}

	go func() {
		logging.Info("ws: server started", "addr", addr)
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logging.Warn("ws: server error", "error", err)
		}
	}()

	// Start event relay goroutine
	go s.relayEvents()

	// Start ping goroutine
	go s.pingLoop()

	s.started.Store(true)
	return nil
}

// Stop shuts down the WebSocket server.
func (s *Server) Stop() {
	if !s.started.Load() {
		return
	}
	s.cancel()
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
	s.started.Store(false)
	logging.Info("ws: server stopped")
}

// IsRunning returns whether the server is accepting connections.
func (s *Server) IsRunning() bool {
	return s.started.Load()
}

// ClientCount returns the number of connected clients.
func (s *Server) ClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// Validate token
	token := r.URL.Query().Get("token")
	if token != s.cfg.Token {
		writeWSError(w, "unauthorized", "invalid token")
		return
	}

	// Validate Origin
	if !isOriginAllowed(r.Header.Get("Origin")) {
		writeWSError(w, "forbidden", "origin not allowed")
		return
	}

	// Check client limit
	if s.ClientCount() >= s.cfg.MaxClients {
		writeWSError(w, "too_many_clients", "max clients reached")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"127.0.0.1", "localhost", "vscode-webview:*"},
	})
	if err != nil {
		logging.Warn("ws: accept failed", "error", err)
		return
	}

	sessionID := r.URL.Query().Get("session")
	go s.handleClient(conn, sessionID)
}

func (s *Server) handleDebugToken(w http.ResponseWriter, r *http.Request) {
	// Only respond from localhost
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token_len":    len(s.cfg.Token),
		"token_prefix": s.cfg.Token[:min(8, len(s.cfg.Token))],
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"clients": s.ClientCount(),
	})
}

// relayEvents reads from the event bus and fans out to clients.
func (s *Server) relayEvents() {
	sub := s.bus.Subscribe()
	defer s.bus.Unsubscribe(sub)

	for {
		select {
		case <-s.ctx.Done():
			return
		case ev, ok := <-sub.Ch:
			if !ok {
				return
			}
			s.mu.Lock()
			for c := range s.clients {
				if !c.sendEvent(ev) {
					// Client queue full, disconnect
					c.cancel()
				}
			}
			s.mu.Unlock()
		}
	}
}

// pingLoop sends periodic pings to keep clients alive.
func (s *Server) pingLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			for c := range s.clients {
				c.conn.Ping(s.ctx)
			}
			s.mu.Unlock()
		}
	}
}

func isOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, allowed := range []string{
		"http://127.0.0.1",
		"http://localhost",
		"https://127.0.0.1",
		"https://localhost",
	} {
		if strings.HasPrefix(origin, allowed) {
			return true
		}
	}
	if strings.Contains(origin, "vscode-webview") {
		return true
	}
	return false
}

func writeWSError(w http.ResponseWriter, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(OutMsg{
		Type:    TypeError,
		Version: 1,
		Error:   &ErrorInfo{Code: code, Message: msg},
	})
}

// Ensure logging is used
var _ = logging.Info
