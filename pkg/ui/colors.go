package ui

import "os"

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

//nolint:gochecknoglobals // process-wide color toggle, set once at startup via SetNoColor
var noColor bool

func SetNoColor(v bool) {
	noColor = v
}

func DisableIfNotTerminal() {
	stat, _ := os.Stdout.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		noColor = true
	}
}

func Red(text string) string {
	if noColor {
		return text
	}
	return colorRed + text + colorReset
}

func Green(text string) string {
	if noColor {
		return text
	}
	return colorGreen + text + colorReset
}

func Yellow(text string) string {
	if noColor {
		return text
	}
	return colorYellow + text + colorReset
}

func Cyan(text string) string {
	if noColor {
		return text
	}
	return colorCyan + text + colorReset
}

func Teal(text string) string {
	if noColor {
		return text
	}
	return colorTeal + text + colorReset
}

func Dim(text string) string {
	if noColor {
		return text
	}
	return colorDim + text + colorReset
}

func Bold(text string) string {
	if noColor {
		return text
	}
	return colorBold + text + colorReset
}
