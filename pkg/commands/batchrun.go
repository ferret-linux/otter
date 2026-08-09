package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ferret-linux/otter/pkg/ui"
)

// resolveContainerNames implements the name-resolution shared by the batch
// commands whose --all behavior is a plain "every container, no filtering":
// either every existing container (all=true) or the explicit names given on
// the command line, erroring if neither was supplied.
//
// pause.go and upgrade.go do not use this helper: pause needs to track
// which listed containers are non-running without excluding them from the
// batch, and upgrade's --running flag needs to exclude them outright. Both
// are genuine behavioral differences, not incidental duplication, so they
// keep their own resolution logic.
func resolveContainerNames(ctx context.Context, listCmd *ListCommand, explicit []string, all bool) ([]string, error) {
	switch {
	case all:
		listResult, err := listCmd.Execute(ctx, ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}
		if len(listResult.Containers) == 0 {
			return nil, ErrNoContainersFound
		}
		names := make([]string, 0, len(listResult.Containers))
		for _, container := range listResult.Containers {
			names = append(names, container.Name)
		}
		return names, nil
	case len(explicit) > 0:
		return explicit, nil
	default:
		return nil, errors.New("please specify a container name or use --all")
	}
}

// batchOutcome tracks the result of running an action across multiple
// container names: which succeeded, which failed, and how many were
// skipped. Skips are never counted as failures.
type batchOutcome struct {
	Succeeded []string
	Failed    []string
	Skipped   int
}

// batchItemFunc runs the action for a single container name. It returns
// skipped=true when the container was intentionally not acted on (e.g.
// already in the target state, or blocked by a precondition), and err when
// the action itself failed. Per-item logging (Ok/Warn/Error) is the
// responsibility of the supplied function.
type batchItemFunc func(ctx context.Context, name string) (skipped bool, err error)

// runBatch runs fn for every name, continuing through failures so every
// name is attempted exactly once, and aggregates the results.
func runBatch(ctx context.Context, names []string, fn batchItemFunc) batchOutcome {
	var outcome batchOutcome
	for _, name := range names {
		skipped, err := fn(ctx, name)
		switch {
		case skipped:
			outcome.Skipped++
		case err != nil:
			outcome.Failed = append(outcome.Failed, name)
		default:
			outcome.Succeeded = append(outcome.Succeeded, name)
		}
	}
	return outcome
}

// batchSummaryConfig controls the wording and exit semantics of the final
// summary line produced by summarizeBatch.
type batchSummaryConfig struct {
	// PastVerb is used in success/failure count messages, e.g. "started",
	// "locked", "restarted".
	PastVerb string
	// BaseVerb is used in "failed to X: name1, name2" messages, e.g.
	// "start", "lock", "restart".
	BaseVerb string
	// AllSkippedMessage is logged when every name was skipped and none
	// succeeded. If empty, a generic "already <PastVerb>" message is used.
	AllSkippedMessage string
	// AllSkippedIsError controls whether an all-skipped, zero-succeeded run
	// is treated as a satisfied no-op (false, exit 0) or as the requested
	// action never having happened (true, exit 1).
	AllSkippedIsError bool
}

// summarizeBatch logs the final summary line for a batch run and returns
// the error Execute should return (nil for success).
func summarizeBatch(outcome batchOutcome, cfg batchSummaryConfig) error {
	total := len(outcome.Succeeded) + len(outcome.Failed) + outcome.Skipped
	succeeded := len(outcome.Succeeded)
	failed := len(outcome.Failed)

	skipNote := ""
	if outcome.Skipped > 0 {
		skipNote = fmt.Sprintf(" (%d skipped)", outcome.Skipped)
	}

	switch {
	case failed > 0:
		failMsg := fmt.Sprintf("failed to %s: %s", cfg.BaseVerb, strings.Join(outcome.Failed, ", "))
		switch {
		case succeeded > 0:
			failMsg = fmt.Sprintf("%d/%d containers %s, %s%s", succeeded, total, cfg.PastVerb, failMsg, skipNote)
		case outcome.Skipped > 0:
			failMsg += skipNote
		}
		return errors.New(failMsg)
	case succeeded == 0 && outcome.Skipped > 0:
		msg := cfg.AllSkippedMessage
		if msg == "" {
			msg = fmt.Sprintf("all %d containers already %s, nothing to do", total, cfg.PastVerb)
		}
		if cfg.AllSkippedIsError {
			return errors.New(msg)
		}
		ui.DefaultLogger.Info(msg)
		return nil
	case outcome.Skipped == 0:
		ui.DefaultLogger.Info(fmt.Sprintf("%s all %d containers", cfg.PastVerb, total))
		return nil
	default:
		ui.DefaultLogger.Info(fmt.Sprintf("%d/%d containers %s%s", succeeded, total, cfg.PastVerb, skipNote))
		return nil
	}
}
