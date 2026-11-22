package ttyutil

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsTerminalWriter reports whether w is an *os.File pointing at an
// interactive terminal. Non-*os.File writers (e.g. buffers in tests) and
// redirected files/pipes both report false.
func IsTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && IsTerminal(f)
}

// IsInteractive reports whether the process's real stderr is an
// interactive terminal. Unlike a Progress's own writer (which callers may
// point at io.Discard to silence status messages), this checks the real
// terminal independently, so callers can show terminal-only UI (like a
// live-updating box) even while Progress itself stays silent.
func IsInteractive() bool {
	return IsTerminal(os.Stderr)
}

// TerminalWidth returns the terminal width of w when w is an interactive
// terminal. It returns 0 for non-terminal writers and on size errors, so
// callers can leave unpaged output unchanged.
func TerminalWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return width
}
