// Package ttyutil provides shared terminal/TTY detection helpers used
// across the codebase (color output, progress rendering, pty allocation).
package ttyutil

import "os"

// IsTerminal reports whether f is an interactive terminal (character device).
func IsTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}
