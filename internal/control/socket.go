// Package control is the IPC boundary between the running easy-alarms GUI and
// the alarmctl CLI / MCP server. It holds the Unix-socket HTTP server, the
// matching client, the wire DTOs and the input parsing/validation — all in one
// package so the two sides of the protocol can never drift apart.
package control

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrAlreadyRunning is returned by Listen when another process already answers
// on the control socket. The GUI treats this as "run without a control server"
// rather than a fatal error.
var ErrAlreadyRunning = errors.New("control socket already in use by another instance")

// SocketPath returns the control socket location:
// $XDG_RUNTIME_DIR/easy-alarms/control.sock, falling back to a per-uid dir
// under the system temp dir when XDG_RUNTIME_DIR is unset.
func SocketPath() string {
	return filepath.Join(socketDir(), "control.sock")
}

func socketDir() string {
	if run := os.Getenv("XDG_RUNTIME_DIR"); run != "" {
		return filepath.Join(run, "easy-alarms")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("easy-alarms-%d", os.Getuid()))
}

// Listen creates the socket directory (0700), clears a stale socket file and
// starts listening with the socket locked down to the owner (0600). If a live
// instance already holds the socket it returns ErrAlreadyRunning.
func Listen() (net.Listener, error) {
	path := SocketPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		// Something is at the path. If it answers, another instance owns it;
		// otherwise it is a leftover from a crash and we reclaim it.
		if c, err := net.DialTimeout("unix", path, 500*time.Millisecond); err == nil {
			c.Close()
			return nil, ErrAlreadyRunning
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	return ln, nil
}
