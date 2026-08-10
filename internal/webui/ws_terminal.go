package webui

import (
	"context"
	"encoding/json"

	"github.com/coder/websocket"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// resizeMessage is the JSON control message the terminal page sends (as a
// websocket text message) on window resize. Terminal input itself is sent
// as binary messages, so the two can share one connection without a custom
// framing protocol — text vs binary is enough.
type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// wsTerminalReader is an io.Reader over a websocket connection that only
// ever returns terminal input (binary messages). Text messages are parsed
// as resizeMessage and pushed onto resizeCh instead of being returned as
// data; anything that doesn't parse as a resize message is silently
// dropped, so this only ever needs one message type going forward.
type wsTerminalReader struct {
	ctx      context.Context //nolint:containedctx // reused across many Read calls; passing it in would change the io.Reader signature
	conn     *websocket.Conn
	resizeCh chan<- containermanager.WinSize
	buf      []byte
}

func newWSTerminalReader(ctx context.Context, conn *websocket.Conn, resizeCh chan<- containermanager.WinSize) *wsTerminalReader {
	return &wsTerminalReader{ctx: ctx, conn: conn, resizeCh: resizeCh}
}

func (r *wsTerminalReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		msgType, data, err := r.conn.Read(r.ctx)
		if err != nil {
			return 0, err
		}

		if msgType == websocket.MessageText {
			var msg resizeMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
				select {
				case r.resizeCh <- containermanager.WinSize{Rows: msg.Rows, Cols: msg.Cols}:
				default:
					// Resize consumer is busy; drop this one, another will
					// follow shortly since the browser resends on settle.
				}
			}
			continue
		}

		r.buf = data
	}

	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}
