package acp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Transport is the framed read/write interface the Client speaks. The default
// implementation reads and writes newline-delimited JSON over an io.Reader /
// io.Writer pair (typically a subprocess's stdout / stdin). Tests inject a
// pipe-backed implementation to exercise the Client without a real subprocess.
//
// Implementations must be safe for one concurrent reader and one concurrent
// writer — never multiple writers (the Client serializes writes internally).
type Transport interface {
	ReadFrame() (*Frame, error)
	WriteFrame(any) error
	Close() error
}

// NewlineTransport implements Transport over newline-delimited JSON-RPC.
// Verified against claude-agent-acp@0.37.0 (the live agent rejects LSP-style
// Content-Length framing with a JSON parse error). One JSON object per line,
// no whitespace inside the line, terminated by '\n'.
type NewlineTransport struct {
	r        *bufio.Reader
	w        io.Writer
	closer   io.Closer
	writeMu  sync.Mutex
}

// NewNewlineTransport builds a transport over an arbitrary reader/writer
// pair. If closer is non-nil, Close() will close it (typically the subprocess
// stdin so the agent observes EOF and exits cleanly).
func NewNewlineTransport(r io.Reader, w io.Writer, closer io.Closer) *NewlineTransport {
	return &NewlineTransport{
		r:      bufio.NewReaderSize(r, 64*1024),
		w:      w,
		closer: closer,
	}
}

// ReadFrame consumes one '\n'-terminated JSON document from the underlying
// reader. Returns io.EOF when the reader closes; the Client treats that as
// the agent having exited.
func (t *NewlineTransport) ReadFrame() (*Frame, error) {
	for {
		// ReadBytes returns the delimiter as part of the slice — fine for the JSON
		// parser, which tolerates trailing whitespace. We use ReadBytes (not Scanner)
		// to avoid Scanner's default 64KB token limit; ACP responses with large tool
		// results can exceed that, and growing Scanner's buffer is more awkward than
		// just sizing the bufio.Reader appropriately.
		line, err := t.r.ReadBytes('\n')
		if err != nil {
			// Surface a partial line if the agent died mid-write. bufio.Reader
			// returns the unterminated bytes alongside io.EOF.
			if len(line) > 0 && errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("acp transport: unexpected EOF mid-frame (%d bytes): %w", len(line), err)
			}
			return nil, err
		}
		if len(line) == 1 {
			// Stray blank line — tolerate by looping (NOT recursing: a buggy
			// host emitting a long run of blank lines would otherwise blow the
			// goroutine stack and take the whole TUI down).
			continue
		}
		var f Frame
		if err := json.Unmarshal(line, &f); err != nil {
			return nil, fmt.Errorf("acp transport: malformed frame: %w (bytes=%d)", err, len(line))
		}
		if f.JSONRPC != "2.0" {
			return nil, fmt.Errorf("acp transport: unexpected jsonrpc version %q", f.JSONRPC)
		}
		return &f, nil
	}
}

// WriteFrame serialises v (typically a Request) as a JSON document and emits
// it followed by '\n'. Serialisation and write are inside a mutex so multiple
// goroutines can call WriteFrame concurrently without corrupting framing.
func (t *NewlineTransport) WriteFrame(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("acp transport: marshal: %w", err)
	}
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if _, err := t.w.Write(data); err != nil {
		return fmt.Errorf("acp transport: write payload: %w", err)
	}
	if _, err := t.w.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("acp transport: write framing: %w", err)
	}
	return nil
}

// Close releases the underlying resources, signalling EOF to the agent.
// Safe to call multiple times; only the first call has effect.
func (t *NewlineTransport) Close() error {
	if t.closer == nil {
		return nil
	}
	err := t.closer.Close()
	t.closer = nil
	return err
}
