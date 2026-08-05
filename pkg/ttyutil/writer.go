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
