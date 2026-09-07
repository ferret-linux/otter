package ttyutil

import (
	"io"
	"os"
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
