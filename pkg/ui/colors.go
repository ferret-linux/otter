package ui

import (
	"os"
	"sync"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorTeal   = "\033[96m"
	colorDim    = "\033[37m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

//nolint:gochecknoglobals // process-wide color toggle, memoized once via NoColor/SetNoColor
var (
	noColorOnce   sync.Once
	noColorValue  bool
	noColorForced bool
)

// SetNoColor explicitly forces the no-color state, overriding the lazy
// terminal/env detection performed by NoColor.
func SetNoColor(v bool) {
	noColorValue = v
	noColorForced = true
}

// ShouldDisableColor reports whether color output should be disabled for
// writes to f: either NO_COLOR is set, TERM=dumb, or f is not a terminal.
func ShouldDisableColor(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return true
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

// NoColor reports whether color output should be suppressed. Unless
// SetNoColor has explicitly forced a value, it lazily computes and caches
// ShouldDisableColor(os.Stdout) on first use, so callers anywhere in the
// program (including ones that run before any explicit setup) converge on
// the same single decision.
func NoColor() bool {
	if !noColorForced {
		noColorOnce.Do(func() {
			noColorValue = ShouldDisableColor(os.Stdout)
		})
	}
	return noColorValue
}

func Red(text string) string {
	if NoColor() {
		return text
	}
	return colorRed + text + colorReset
}

func Green(text string) string {
	if NoColor() {
		return text
	}
	return colorGreen + text + colorReset
}

func Yellow(text string) string {
	if NoColor() {
		return text
	}
	return colorYellow + text + colorReset
}

func Cyan(text string) string {
	if NoColor() {
		return text
	}
	return colorCyan + text + colorReset
}

func Teal(text string) string {
	if NoColor() {
		return text
	}
	return colorTeal + text + colorReset
}

func Dim(text string) string {
	if NoColor() {
		return text
	}
	return colorDim + text + colorReset
}

func Bold(text string) string {
	if NoColor() {
		return text
	}
	return colorBold + text + colorReset
}
