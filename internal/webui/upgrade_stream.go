package webui

import (
	"strings"
	"sync"
)

// upgradeStream buffers a single upgrade run's output and fans it out to
// zero or more attached viewers. It exists because the upgrade itself runs
// detached from any one HTTP request (see server.startUpgrade), so no
// single connection owns its output; viewers attach and detach
// independently of whether the upgrade is still running.
type upgradeStream struct {
	mu   sync.Mutex
	buf  []string      // lines written so far, for viewers that attach late
	subs []chan string // live viewers; each Write fans out to every channel
	done bool
}

// newUpgradeStream returns an empty, live stream ready to accept writes and
// subscribers.
func newUpgradeStream() *upgradeStream {
	return &upgradeStream{}
}

// write appends a line to the buffer and fans it out to every current
// subscriber. Subscribers that are not keeping up are skipped for this
// line rather than blocking the upgrade itself.
func (u *upgradeStream) write(line string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.buf = append(u.buf, line)
	for _, ch := range u.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

// close marks the stream as finished and closes every subscriber channel,
// so attached viewers know the upgrade has ended.
func (u *upgradeStream) close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.done = true
	for _, ch := range u.subs {
		close(ch)
	}
	u.subs = nil
}

// subscribe attaches a new viewer, returning the lines written so far and a
// channel that receives subsequent lines. The channel is already closed if
// the stream has finished. Callers must call unsubscribe when done reading,
// unless the returned channel was already closed.
func (u *upgradeStream) subscribe() ([]string, chan string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	buf := append([]string(nil), u.buf...)

	ch := make(chan string, 16)
	if u.done {
		close(ch)
		return buf, ch
	}

	u.subs = append(u.subs, ch)
	return buf, ch
}

// unsubscribe detaches a viewer's channel, so writes after this point no
// longer attempt to send to it.
func (u *upgradeStream) unsubscribe(ch chan string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for i, c := range u.subs {
		if c == ch {
			u.subs = append(u.subs[:i], u.subs[i+1:]...)
			return
		}
	}
}

// upgradeStreamWriter adapts an *upgradeStream to io.Writer, splitting
// writes on newlines the same way sseLogWriter does for logs, since
// commands.UpgradeOptions.Stdout/Stderr are plain io.Writers and upgrade
// output arrives in arbitrarily-chunked multi-line writes.
type upgradeStreamWriter struct {
	stream *upgradeStream
}

func (w *upgradeStreamWriter) Write(p []byte) (int, error) {
	trimmed := strings.TrimRight(string(p), "\n")
	if trimmed == "" {
		return len(p), nil
	}

	for _, line := range strings.Split(trimmed, "\n") {
		w.stream.write(line)
	}
	return len(p), nil
}
