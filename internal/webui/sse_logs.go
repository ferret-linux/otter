package webui

import (
	"fmt"
	"net/http"
	"strings"
)

// sseLogTailLines caps how much log history a fresh /sse/containers/{name}/logs
// connection replays before switching to live-follow, so opening the logs
// page doesn't dump a container's entire history into the browser.
const sseLogTailLines = 200

// sseLogWriter is an io.Writer over an http.ResponseWriter already committed
// to text/event-stream, used as Journal's Stdout/Stderr so each chunk of log
// output is framed as an SSE message and flushed immediately instead of
// buffering. Journal writes multi-line chunks in a single Write call, but
// SSE "data:" fields can't contain literal newlines, so each line is sent as
// its own "data:" field within one message.
type sseLogWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (s *sseLogWriter) Write(p []byte) (int, error) {
	trimmed := strings.TrimRight(string(p), "\n")
	if trimmed == "" {
		return len(p), nil
	}

	for _, line := range strings.Split(trimmed, "\n") {
		if _, err := fmt.Fprintf(s.w, "data: %s\n", line); err != nil {
			return 0, err
		}
	}
	if _, err := fmt.Fprint(s.w, "\n"); err != nil {
		return 0, err
	}

	s.flusher.Flush()
	return len(p), nil
}
