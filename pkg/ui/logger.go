package ui

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	writer io.Writer
}

func NewLogger(writer io.Writer) *Logger {
	return &Logger{writer: writer}
}

// DefaultLogger writes to stderr
//
//nolint:gochecknoglobals // singleton: process-wide logger instance
var DefaultLogger = NewLogger(os.Stderr)

func (l *Logger) Ok(msg string, a ...any) {
	icon := Green("[✓]")
	fmt.Fprintf(l.writer, "%s %s\n", icon, fmt.Sprintf(msg, a...))
}

func (l *Logger) Error(msg string, a ...any) {
	icon := Red("[✗]")
	text := Red(fmt.Sprintf(msg, a...))
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}

func (l *Logger) Warn(msg string, a ...any) {
	icon := Yellow("[⚠]")
	fmt.Fprintf(l.writer, "%s %s\n", icon, fmt.Sprintf(msg, a...))
}

func (l *Logger) Info(msg string, a ...any) {
	icon := Cyan("[i]")
	text := Dim(fmt.Sprintf(msg, a...))
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}

func (l *Logger) Notice(msg string, a ...any) {
	icon := Teal("[‣]")
	text := Dim(fmt.Sprintf(msg, a...))
	fmt.Fprintf(l.writer, "%s %s\n", icon, text)
}
