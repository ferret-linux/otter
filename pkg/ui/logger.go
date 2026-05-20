package ui

import (
	"fmt"
	"io"
)

type Logger struct {
	writer io.Writer
}

func Logger(writer io.Writer) *Logger {
	return &logger{writer: writer}
}

func (l *Logger) Ok(msg string, a ...any) {
	icon := colorGreen + "[✓]" + colorReset
	fmt.Fprintf(l.writer, "%s %s\n", icon, fmt.Sprintf(msg, a...))
}

func (l *Logger) Error(msg string, a ...any) {
	icon := colorRed + "[✗]" + colorReset
	text := colorRed + fmt.Sprintf(msg, a...) + colorReset
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}

func (l *Logger) Warn(msg string, a ...any) {
	icon := colorYellow + "[⚠]" + colorReset
	fmt.Fprintf(l.writer, "%s %s\n", icon, fmt.Sprintf(msg, a...))
}

func (l *Logger) Info(msg string, a ...any) {
	icon := colorCyan + "[ℹ]" + colorReset
	text := colorDim + fmt.Sprintf(msg, a...) + colorReset
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}

func (l *Logger) Notice(msg string, a ...any) {
	icon := colorTeal + "[‣]" + colorReset
	text := colorDim + fmt.Sprintf(msg, a...) + colorReset
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}
