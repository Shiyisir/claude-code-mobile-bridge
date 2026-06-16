package stream

import (
	"io"
)

const maxSampleBytes = 64 * 1024 // 64KB

// Tee copies from src to dst while capturing up to maxSampleBytes into buf.
// If sidecar is non-nil, each line is sent non-blockingly to the sidecar channel.
// Primary copy to dst is never delayed by sidecar drops.
func Tee(dst io.Writer, src io.Reader, buf *[]byte, sidecar *Sidecar) (written int64, err error) {
	w := &teeWriter{dst: dst, buf: buf, cap: maxSampleBytes, sidecar: sidecar}
	return io.Copy(w, src)
}

type teeWriter struct {
	dst     io.Writer
	buf     *[]byte
	cap     int
	sidecar *Sidecar
	lineBuf []byte
}

func (w *teeWriter) Write(p []byte) (int, error) {
	// Sample capture (best-effort)
	remaining := w.cap - len(*w.buf)
	if remaining > 0 {
		n := len(p)
		if n > remaining {
			n = remaining
		}
		*w.buf = append(*w.buf, p[:n]...)
	}

	// Non-blocking line extraction for sidecar
	if w.sidecar != nil {
		w.lineBuf = append(w.lineBuf, p...)
		for {
			idx := indexByte(w.lineBuf, '\n')
			if idx < 0 {
				break
			}
			line := w.lineBuf[:idx]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if len(line) > 0 {
				w.sidecar.Send(line)
			}
			w.lineBuf = w.lineBuf[idx+1:]
		}
		// Prevent lineBuf from growing unbounded if no newlines
		if len(w.lineBuf) > 256*1024 {
			w.lineBuf = nil
		}
	}

	// Always write to primary destination — this is the main link
	return w.dst.Write(p)
}

func indexByte(s []byte, c byte) int {
	for i, b := range s {
		if b == c {
			return i
		}
	}
	return -1
}
