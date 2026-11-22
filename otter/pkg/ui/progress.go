package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Progress struct {
	pending     bool
	lastMessage string
	writer      io.Writer
	jsonMode    bool
}

func NewProgress(writer io.Writer) *Progress {
	return &Progress{
		pending: false,
		writer:  writer,
	}
}

func NewJSONProgress() *Progress {
	return &Progress{
		pending:  false,
		jsonMode: true,
	}
}

func NewDevNullProgress() *Progress {
	return &Progress{
		pending: false,
		writer:  io.Discard,
	}
}

func (p *Progress) emitJSON(message string) {
	b, _ := json.Marshal(message)
	fmt.Fprintf(os.Stdout, "{\"message\":%s}\n", b)
}

func (p *Progress) Next(message string, a ...any) {
	p.pending = false

	p.pending = true
	p.lastMessage = fmt.Sprintf(message, a...)

	if p.jsonMode {
		p.emitJSON(p.lastMessage)
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Info(p.lastMessage)
	}
}

func (p *Progress) Finalize(message string, a ...any) {
	p.pending = false
	msg := fmt.Sprintf(message, a...)

	if p.jsonMode {
		p.emitJSON(msg)
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Info(msg)
	}
}

func (p *Progress) Done() {
	p.pending = false
}

func (p *Progress) Fail() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.jsonMode {
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Error(p.lastMessage)
	}
}
