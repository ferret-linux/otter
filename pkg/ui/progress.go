package ui

import (
	"fmt"
	"io"
	"os"
)

type Progress struct {
	pending     bool
	lastMessage string
	writer      io.Writer
}

func NewProgress(writer io.Writer) *Progress {
	return &Progress{
		pending: false,
		writer:  writer,
	}
}

func NewDevNullProgress() *Progress {
	return &Progress{
		pending: false,
		writer:  io.Discard,
	}
}

func (p *Progress) Next(message string, a ...any) {
	if p.pending {
		p.Done()
	}

	p.pending = true
	p.lastMessage = fmt.Sprintf(message, a...)

	if p.writer != io.Discard {
		DefaultLogger.Info("%s", p.lastMessage)
	}
}

func (p *Progress) Finalize(message string, a ...any) {
	p.Done()
	if p.writer != io.Discard {
		DefaultLogger.Ok("%s", fmt.Sprintf(message, a...))
	}
}

// isTerminalWriter reports whether w is an *os.File pointing at an
// interactive terminal. Non-*os.File writers (e.g. buffers in tests) and
// redirected files/pipes both report false.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isTerminal(f)
}

func (p *Progress) Done() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.writer != io.Discard {
		if isTerminalWriter(p.writer) {
			fmt.Fprintf(p.writer, "\033[1A\r\033[2K")
		}
		DefaultLogger.Ok("%s", p.lastMessage)
	}
}

func (p *Progress) Fail() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.writer != io.Discard {
		DefaultLogger.Error("%s", p.lastMessage)
	}
}
