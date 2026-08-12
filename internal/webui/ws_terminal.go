package webui

import (
	"context"
	"encoding/json"

	"github.com/coder/websocket"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// clientMessage is the JSON envelope for every text-message a browser tab
// sends up a session websocket. Terminal input itself is sent as binary
// messages, so the two never collide on one connection without a custom
// framing protocol — text vs binary is enough, same as the original
// resize-only protocol this replaces.
type clientMessage struct {
	Type string `json:"type"`
	// Cols and Rows are set for Type == "resize".
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
	// Accept is set for Type == "resolve_request".
	Accept bool `json:"accept,omitempty"`
}

// runSessionReadLoop reads every message sent up conn for the lifetime of
// the connection (until it errors or the context is canceled) and applies
// it to sess/c: binary messages are forwarded as terminal input (subject
// to sess.input's own controller check), and text messages are dispatched
// by clientMessage.Type — "resize" pushes onto resizeCh, "request_control"
// asks sess for control, "resolve_request" answers a pending request this
// connection (as controller) received.
//
// This is shared by both terminalWS and upgradeWS, since both are now
// session-backed the same way.
func runSessionReadLoop(ctx context.Context, conn *websocket.Conn, sess *session, c *sessionConn, resizeCh chan<- containermanager.WinSize) error {
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}

		if msgType == websocket.MessageBinary {
			sess.input(c, data)
			continue
		}

		var msg clientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue // not a message we understand; ignore rather than fail the connection
		}

		switch msg.Type {
		case "resize":
			select {
			case resizeCh <- containermanager.WinSize{Rows: msg.Rows, Cols: msg.Cols}:
			default:
				// Resize consumer is busy; drop this one, another will
				// follow shortly since the browser resends on settle.
			}
		case "request_control":
			sess.requestControl(c)
		case "resolve_request":
			sess.resolveRequest(c, msg.Accept)
		}
	}
}

// runSessionWriteLoop writes every message queued on c.send to conn until
// the channel is closed (by session.detach or session.close) or a write
// fails. It owns all writes to conn so runSessionReadLoop's goroutine and
// the session's fan-out never write to the same *websocket.Conn
// concurrently.
func runSessionWriteLoop(ctx context.Context, conn *websocket.Conn, c *sessionConn) {
	for msg := range c.send {
		if conn.Write(ctx, websocket.MessageText, msg) != nil {
			return
		}
	}
}
