package stream

// Sidecar is a non-blocking line delivery channel.
// When the buffer is full, lines are dropped (counted) — main link is never blocked.
type Sidecar struct {
	ch      chan []byte
	dropped int64
}

// NewSidecar creates a Sidecar with the given buffer capacity.
func NewSidecar(bufSize int) *Sidecar {
	return &Sidecar{ch: make(chan []byte, bufSize)}
}

// Send tries to send a line. Returns false if dropped.
func (s *Sidecar) Send(line []byte) bool {
	// Copy the line so the caller can reuse the buffer
	cp := make([]byte, len(line))
	copy(cp, line)
	select {
	case s.ch <- cp:
		return true
	default:
		s.dropped++
		return false
	}
}

// Recv returns the receive channel.
func (s *Sidecar) Recv() <-chan []byte {
	return s.ch
}

// Dropped returns the count of dropped lines.
func (s *Sidecar) Dropped() int64 {
	return s.dropped
}
