package ui

import (
	"fmt"
	"io"

	"github.com/ferret-linux/otter/pkg/ttyutil"
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
		DefaultLogger.Info(p.lastMessage)
	}
}

func (p *Progress) Finalize(message string, a ...any) {
	p.Done()
	if p.writer != io.Discard {
		DefaultLogger.Info(fmt.Sprintf(message, a...))
	}
}

func (p *Progress) Done() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.writer != io.Discard {
		if ttyutil.IsTerminalWriter(p.writer) {
			fmt.Fprintf(p.writer, "\033[1A\r\033[2K")
		}
		DefaultLogger.Info(p.lastMessage)
	}
}

func (p *Progress) Fail() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.writer != io.Discard {
		DefaultLogger.Error(p.lastMessage)
	}
}
