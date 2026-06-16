package parser

import (
	"sync/atomic"

	"github.com/shiyisir/claude-proxy/internal/logging"
	"github.com/shiyisir/claude-proxy/internal/stream"
)

// Config holds parser options.
type Config struct {
	MaxRawBytes int  // max bytes for raw event field (default 8192)
	DropUnknown bool // drop events with kind=unknown (default true)
}

// DefaultConfig returns the default parser config.
func DefaultConfig() Config {
	return Config{
		MaxRawBytes: 8192,
		DropUnknown: true,
	}
}

// Parser reads lines from a sidecar channel, parses NDJSON, and invokes callbacks.
type Parser struct {
	cfg           Config
	source        string // "stdout" or "stderr"
	onEvent       Callback
	failCount     atomic.Int32
	parsed        atomic.Int32
	droppedUnknown atomic.Int32
	stopped       atomic.Bool
}

// Callback is invoked for each normalized event. Must not block.
type Callback func(*NormalizedEvent)

// New creates a new Parser.
func New(cfg Config, source string, onEvent Callback) *Parser {
	return &Parser{cfg: cfg, source: source, onEvent: onEvent}
}

// Run reads lines from the sidecar until stopped. Never blocks the caller.
func (p *Parser) Run(sidecar *stream.Sidecar) {
	for !p.stopped.Load() {
		line, ok := <-sidecar.Recv()
		if !ok {
			return
		}
		if len(line) == 0 {
			continue
		}

		// Skip lines that aren't valid JSON (don't write as unknown)
		raw := RawFromLine(line)
		if raw == nil {
			fc := p.failCount.Add(1)
			if fc > 500 {
				logging.Warn("parser: too many non-JSON lines, stopping", "source", p.source)
				p.Stop()
			}
			continue
		}
		p.failCount.Store(0)

		event := Normalize(p.cfg, p.source, raw)

		// Log first few events for diagnostics
		if n := p.parsed.Add(1); n <= 5 {
			logging.Info("parser: event", "kind", event.Kind, "session_id", event.SessionID, "raw_type", event.RawType)
		}

		// Skip unknown events if configured
		if p.cfg.DropUnknown && event.Kind == KindUnknown {
			p.droppedUnknown.Add(1)
			continue
		}

		if p.onEvent != nil {
			p.onEvent(event)
		}
	}
	// Log dropped count on stop
	if n := sidecar.Dropped(); n > 0 {
		logging.Info("parser: sidecar drops", "source", p.source, "dropped", n)
	}
}

// Stats returns parser statistics.
func (p *Parser) Stats() (parsed, droppedUnknown int32) {
	return p.parsed.Load(), p.droppedUnknown.Load()
}

// Stop signals the parser to stop reading.
func (p *Parser) Stop() {
	p.stopped.Store(true)
}
