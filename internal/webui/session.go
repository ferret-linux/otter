package webui

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// controlRequestTimeout is how long a session waits for the current
// controller to respond to a control request before auto-granting it to
// the requester, per the product decision that a stuck/absent controller
// should not be able to block handoff indefinitely.
const controlRequestTimeout = 5 * time.Second

// sessionConn is one attached websocket's view of a session: every
// connection (controller or viewer) gets one of these so the session can
// address it individually (send it output, send it control-state changes,
// send it "someone wants control").
type sessionConn struct {
	id   uint64
	conn *websocket.Conn
	// send serializes writes to conn; websocket.Conn is not safe for
	// concurrent writes, and both the session's fan-out and this
	// connection's own control-state messages write to it.
	send chan []byte
}

// session is a single running interactive process (a shell entered via
// `otter enter`, or an `otter-init --upgrade` run) shared by every browser
// tab attached to it. Exactly one attached connection is ever "in control"
// (its keystrokes reach the process's stdin); every other attached
// connection is a view-only viewer receiving the same output. Control can
// move between connections via requestControl.
//
// A session outlives any single connection: for a shell, it lives as long
// as at least one connection is attached; for an upgrade, it lives for the
// lifetime of the upgrade regardless of whether anyone is attached at all
// (see server.upgradeAction), matching the existing behaviour that closing
// the browser must not cancel an in-progress upgrade.
type session struct {
	mu sync.Mutex

	// buf holds every line written so far, for connections that attach
	// after output has already started.
	buf []string
	// conns is every currently attached connection, controller included.
	conns map[uint64]*sessionConn
	// controllerID is the id of the connection currently in control, or 0
	// if no connection has ever attached (can't happen once at least one
	// has, since attaching with no existing controller grants control
	// immediately).
	controllerID uint64
	nextConnID   uint64

	// stdinW is where the current controller's input bytes are written.
	// The process's actual Stdin reader is stdinR (see newSession); stdinW
	// is redirected internally, not reassigned, so the process's reader
	// never needs to know control changed hands.
	stdinW io.Writer

	// pendingRequesterID is set while a control request is awaiting the
	// current controller's response; 0 when no request is pending.
	pendingRequesterID uint64
	pendingTimer       *time.Timer

	done bool
}

// sessionIO is what a session hands back for wiring into the underlying
// process: Stdin should be passed as the process's stdin, and the session
// itself should be used as the process's stdout/stderr writer (via Write,
// making *session an io.Writer).
type sessionIO struct {
	Stdin io.Reader
}

// newSession creates an empty, live session and the stdin reader that
// should be handed to the underlying process (Enter's opts.Stdin). The
// reader blocks for input until a controller is attached and writes to it,
// exactly like a real terminal blocks until someone types.
func newSession() (*session, sessionIO) {
	r, w := io.Pipe()
	s := &session{
		conns:  make(map[uint64]*sessionConn),
		stdinW: w,
	}
	return s, sessionIO{Stdin: r}
}

// Write implements io.Writer so *session can be passed directly as a
// process's Stdout/Stderr. It buffers the line(s) and fans them out to
// every attached connection, splitting on newlines the same way the old
// upgradeStreamWriter did, since output arrives in arbitrarily-chunked
// multi-line writes.
func (s *session) Write(p []byte) (int, error) {
	trimmed := strings.TrimRight(string(p), "\n")
	if trimmed == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(trimmed, "\n") {
		s.broadcastOutput(line)
	}
	return len(p), nil
}

func (s *session) broadcastOutput(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf = append(s.buf, line)
	msg, _ := json.Marshal(wireMessage{Type: "output", Data: line})
	for _, c := range s.conns {
		s.sendLocked(c, msg)
	}
}

// sendLocked queues msg for c without blocking the caller; if c isn't
// keeping up, the message is dropped for that connection rather than
// stalling every other connection's output, matching upgradeStream.write's
// original non-blocking fan-out tradeoff.
func (s *session) sendLocked(c *sessionConn, msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

// close marks the session finished and disconnects every attached
// connection. Called once the underlying process (shell or upgrade) exits.
func (s *session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return
	}
	s.done = true
	if s.pendingTimer != nil {
		s.pendingTimer.Stop()
	}
	msg, _ := json.Marshal(wireMessage{Type: "closed"})
	for _, c := range s.conns {
		s.sendLocked(c, msg)
		close(c.send)
	}
	s.conns = nil
}

// attach registers a new connection with the session. The first connection
// to attach becomes controller automatically; later connections attach as
// viewers. Returns the connection handle and the buffered output so far,
// so the caller can replay it to the new connection before live output
// resumes.
func (s *session) attach(conn *websocket.Conn) (*sessionConn, []string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return nil, nil, false
	}

	s.nextConnID++
	c := &sessionConn{id: s.nextConnID, conn: conn, send: make(chan []byte, 32)}
	s.conns[c.id] = c

	if s.controllerID == 0 {
		s.controllerID = c.id
	}

	buf := append([]string(nil), s.buf...)
	return c, buf, true
}

// detach removes a connection from the session. If it was the controller,
// control passes to an arbitrary remaining connection, if any; the process
// itself is left running either way (see session doc comment).
func (s *session) detach(c *sessionConn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conns, c.id)
	close(c.send)

	if s.pendingRequesterID == c.id {
		s.clearPendingLocked()
	}

	if s.controllerID != c.id {
		return
	}
	s.controllerID = 0
	for id := range s.conns {
		s.controllerID = id
		break
	}
	s.broadcastControlStateLocked()
}

// isController reports whether c currently holds control.
func (s *session) isController(c *sessionConn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.controllerID == c.id
}

// input is called with a chunk of terminal input from c. It is only
// forwarded to the process's stdin if c is currently the controller;
// input from a viewer is silently ignored, since a viewer that wants to
// type must first request and receive control.
func (s *session) input(c *sessionConn, data []byte) {
	s.mu.Lock()
	isController := s.controllerID == c.id
	w := s.stdinW
	s.mu.Unlock()

	if !isController {
		return
	}
	_, _ = w.Write(data)
}

// requestControl is called when connection c asks to become controller. If
// c already is the controller, or no other connection is attached, this is
// a no-op. Otherwise the current controller is notified and given
// controlRequestTimeout to respond (via resolveRequest); if it doesn't,
// control is granted to c automatically.
func (s *session) requestControl(c *sessionConn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controllerID == c.id || s.controllerID == 0 {
		return
	}
	// Only one pending request at a time; a second requester simply waits
	// for the first to resolve rather than pre-empting it.
	if s.pendingRequesterID != 0 {
		return
	}

	current, ok := s.conns[s.controllerID]
	if !ok {
		return
	}

	s.pendingRequesterID = c.id
	msg, _ := json.Marshal(wireMessage{Type: "control_requested"})
	s.sendLocked(current, msg)

	s.pendingTimer = time.AfterFunc(controlRequestTimeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.pendingRequesterID != c.id {
			return // already resolved explicitly
		}
		s.grantControlLocked(c.id)
		s.clearPendingLocked()
	})
}

// resolveRequest is called when the current controller explicitly
// accepts or denies a pending control request. If nothing is pending, or
// c is not the current controller, this is a no-op.
func (s *session) resolveRequest(c *sessionConn, accept bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controllerID != c.id || s.pendingRequesterID == 0 {
		return
	}

	requesterID := s.pendingRequesterID
	s.clearPendingLocked()

	if accept {
		s.grantControlLocked(requesterID)
	}
}

func (s *session) clearPendingLocked() {
	if s.pendingTimer != nil {
		s.pendingTimer.Stop()
		s.pendingTimer = nil
	}
	s.pendingRequesterID = 0
}

func (s *session) grantControlLocked(newControllerID uint64) {
	if _, ok := s.conns[newControllerID]; !ok {
		return
	}
	s.controllerID = newControllerID
	s.broadcastControlStateLocked()
}

// broadcastControlStateLocked tells every attached connection who the
// controller is now, so each browser tab can show itself as either
// interactive or view-only.
func (s *session) broadcastControlStateLocked() {
	for id, c := range s.conns {
		msg, _ := json.Marshal(wireMessage{Type: "control_state", Controller: id == s.controllerID})
		s.sendLocked(c, msg)
	}
}

// wireMessage is the JSON envelope for every message a session sends down
// a websocket. Terminal input from the browser is sent as raw binary
// messages (see wsSessionReader), separately from this JSON control
// channel, the same way resizeMessage and binary input already coexist on
// one connection in ws_terminal.go.
type wireMessage struct {
	Type string `json:"type"`
	// Data carries a line of process output for Type == "output".
	Data string `json:"data,omitempty"`
	// Controller carries this connection's new control state for
	// Type == "control_state".
	Controller bool `json:"controller,omitempty"`
}
