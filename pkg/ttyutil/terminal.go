// Package ttyutil provides shared terminal/TTY detection helpers used
// across the codebase (color output, progress rendering, pty allocation).
package ttyutil

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether f is an interactive terminal.
//
// A char-device test is not enough: /dev/null (and similar redirects) is a
// character device but not a terminal, so a headless invocation piped to
// /dev/null would be misdetected as interactive — wrongly emitting color
// codes/progress bars into redirected output, and (via IsTTY) causing a
// --tty to be allocated on the container engine when it shouldn't be.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
