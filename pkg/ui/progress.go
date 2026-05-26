package ui

import (
	"fmt"
	"io"
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

func (p *Progress) Done() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.writer != io.Discard {
		fmt.Fprintf(p.writer, "\033[1A\r\033[2K")
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
