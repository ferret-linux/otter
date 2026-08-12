package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// notification is one entry in the notification panel: the result of a
// user-triggered action (start/stop/remove, registry pull, settings save,
// manifest apply, ...), or a background failure that has no HTTP request
// to attach an error to (e.g. an async registry pull or manifest apply
// run via goroutine — see registryPullAction, manifestsApply).
type notification struct {
	ID      int       `json:"id"`
	Level   string    `json:"level"` // "error", "success", or "info"
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// notificationEvent is one message sent down /sse/notifications. Type
// discriminates between a new notification ("add"), a single notification
// being dismissed ("delete"), and the whole panel being cleared ("clear"),
// so already-connected viewers stay in sync with hub.buf without having to
// reconnect. A newly-connecting viewer never sees "delete"/"clear" events
// directly — subscribe replays hub.buf (which is already the
// post-deletion state) as a sequence of "add" events instead.
type notificationEvent struct {
	Type         string        `json:"type"` // "add", "delete", or "clear"
	Notification *notification `json:"notification,omitempty"`
	ID           int           `json:"id,omitempty"`
}

// notificationMaxBuffered caps how many past notifications are kept
// in-memory and replayed to a newly-connected /sse/notifications viewer.
// Matches the rest of webui's ephemeral, in-memory-only session state
// (see server.pulling, server.upgradeStreams) — nothing here survives a
// webui restart.
const notificationMaxBuffered = 100

// notificationHub buffers notifications and fans them out to every
// connected /sse/notifications viewer, mirroring upgradeStream's
// buffer-plus-subscriber-channels pattern (see upgrade_stream.go). Unlike
// an upgradeStream, a notificationHub never closes: it lives for the
// whole webui process, not one upgrade run. buf holds the current, live
// set of notifications — delete/clear remove entries from it directly
// rather than just appending — so it doubles as the persisted state a
// fresh subscriber replays.
type notificationHub struct {
	mu     sync.Mutex
	nextID int
	buf    []notification
	subs   []chan notificationEvent
}

// newNotificationHub returns an empty hub ready to accept notifications
// and subscribers.
func newNotificationHub() *notificationHub {
	return &notificationHub{}
}

// notify appends a new notification and fans it out to every current
// subscriber. Subscribers that are not keeping up are skipped for this
// notification rather than blocking the caller — same tradeoff
// upgradeStream.write makes for upgrade output.
func (h *notificationHub) notify(level, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	n := notification{ID: h.nextID, Level: level, Message: message, Time: time.Now()}

	h.buf = append(h.buf, n)
	if len(h.buf) > notificationMaxBuffered {
		h.buf = h.buf[len(h.buf)-notificationMaxBuffered:]
	}

	h.broadcast(notificationEvent{Type: "add", Notification: &n})
}

// delete removes the notification with the given id from buf, if present,
// and tells every current subscriber to drop it too. Deleting an id that
// no longer exists (already dismissed, or a stale/duplicate request) is a
// no-op, not an error.
func (h *notificationHub) delete(id int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, n := range h.buf {
		if n.ID == id {
			h.buf = append(h.buf[:i], h.buf[i+1:]...)
			break
		}
	}

	h.broadcast(notificationEvent{Type: "delete", ID: id})
}

// clear empties buf and tells every current subscriber to clear their
// panel too.
func (h *notificationHub) clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buf = nil
	h.broadcast(notificationEvent{Type: "clear"})
}

// broadcast fans out ev to every current subscriber. Callers must hold
// h.mu. Subscribers that are not keeping up are skipped for this event
// rather than blocking the caller — same tradeoff upgradeStream.write
// makes for upgrade output.
func (h *notificationHub) broadcast(ev notificationEvent) {
	for _, ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// subscribe attaches a new viewer, returning the buffered backlog and a
// channel that receives subsequent notification events. Callers must call
// unsubscribe when done reading.
func (h *notificationHub) subscribe() ([]notification, chan notificationEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	buf := append([]notification(nil), h.buf...)
	ch := make(chan notificationEvent, 16)
	h.subs = append(h.subs, ch)
	return buf, ch
}

// unsubscribe detaches a viewer's channel, so notify/delete/clear after
// this point no longer attempt to send to it.
func (h *notificationHub) unsubscribe(ch chan notificationEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, c := range h.subs {
		if c == ch {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			return
		}
	}
}

// notify is a convenience wrapper so handlers can call s.notify(...)
// directly instead of s.notifications.notify(...).
func (s *server) notify(level, message string) {
	s.notifications.notify(level, message)
}

// writeNotificationEvent writes one notification event as a
// "data: <json>\n\n" SSE message.
func writeNotificationEvent(w http.ResponseWriter, ev notificationEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

// notificationsSSE handles GET /sse/notifications: replays the buffered
// backlog as "add" events, then streams new notification events as they
// happen, matching upgradeSSE's connection handling (see handlers.go) —
// same event-stream headers, same flush-per-message, same
// disconnect-on-context-done, plus the same 30s keepalive comment so idle
// connections aren't dropped by intermediary proxies.
func (s *server) notificationsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	backlog, ch := s.notifications.subscribe()
	defer s.notifications.unsubscribe(ch)

	for _, n := range backlog {
		n := n
		if err := writeNotificationEvent(w, notificationEvent{Type: "add", Notification: &n}); err != nil {
			return
		}
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			if err := writeNotificationEvent(w, ev); err != nil {
				return
			}
			flusher.Flush()
		case <-time.After(30 * time.Second):
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// notificationDeleteAction handles DELETE /notifications/{id}: dismisses
// one notification. The id path value is expected to be the integer
// notification.ID; an unparseable id is a 400, since it can only come from
// a malformed request (the panel always sends the id it was given).
func (s *server) notificationDeleteAction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid notification id", http.StatusBadRequest)
		return
	}
	s.notifications.delete(id)
	w.WriteHeader(http.StatusNoContent)
}

// notificationClearAction handles DELETE /notifications: dismisses every
// notification.
func (s *server) notificationClearAction(w http.ResponseWriter, r *http.Request) {
	s.notifications.clear()
	w.WriteHeader(http.StatusNoContent)
}
