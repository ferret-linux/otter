package ttyutil

import "os"

// IsTTY returns true if both stdin and stdout are terminals.
// Mirrors the shell's: if [ ! -t 0 ] || [ ! -t 1 ]; then headless=1; fi
func IsTTY() bool {
	return IsTerminal(os.Stdin) && IsTerminal(os.Stdout)
}
