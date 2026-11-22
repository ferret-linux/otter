package providers

import "time"

// Sleep durations used while waiting for a container's setup to complete.
const (
	// logsRetryInterval is the first delay before retrying when fetching
	// container logs fails transiently during setup waiting.
	logsRetryInterval = 100 * time.Millisecond
	// setupPollInterval is the cadence at which container setup logs are
	// polled while waiting for the setup-done marker.
	setupPollInterval = 500 * time.Millisecond
	// setupRetryBackoffMax caps log-fetch backoff at the normal poll cadence.
	setupRetryBackoffMax = setupPollInterval
	// logCursorOverlap is subtracted from the next --since cursor so that a
	// log line emitted while a fetch is in flight is still picked up by the
	// following poll instead of falling into a permanent gap.
	logCursorOverlap = 2 * time.Second
)
