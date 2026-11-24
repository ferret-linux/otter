package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

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
	// defaultDockerSetupTimeout bounds the Docker setup watcher so a stalled
	// in-container helper fails with a clear message instead of waiting until
	// an outer CLI timeout cancels the context.
	defaultDockerSetupTimeout = 5 * time.Minute
)

// dockerSetupTimeout returns the bounded setup deadline. The environment
// override is useful for diagnosing slow images; invalid values are ignored.
func dockerSetupTimeout() time.Duration {
	if raw := os.Getenv("OTTER_DOCKER_SETUP_TIMEOUT"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultDockerSetupTimeout
}

// setupWaitError turns cancellation into a readable setup failure, with a
// dedicated message for the setup deadline.
func setupWaitError(ctx context.Context, err error, timeout time.Duration) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("container setup did not finish within %s", timeout)
	}
	if err == nil {
		err = ctx.Err()
	}
	return fmt.Errorf("docker: %w", err)
}

// sleepContext waits for d, or returns ctx.Err() if the context is cancelled
// first, so setup watchers stay responsive to cancellation.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait interrupted: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
