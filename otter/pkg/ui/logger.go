package ui

import (
	"os"

	"charm.land/log/v2"
)

// DefaultLogger writes to stderr.
//
//nolint:gochecknoglobals // singleton: process-wide logger instance
var DefaultLogger = log.New(os.Stderr)
