package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ferret-linux/otter/pkg/ttyutil"
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
	if p.pending {
		p.Done()
	}

	p.pending = true
	p.lastMessage = fmt.Sprintf(message, a...)

	if p.jsonMode {
		p.emitJSON(p.lastMessage)
		DefaultLogger.Info(p.lastMessage)
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Info(p.lastMessage)
	}
}

func (p *Progress) Finalize(message string, a ...any) {
	p.Done()
	msg := fmt.Sprintf(message, a...)

	if p.jsonMode {
		p.emitJSON(msg)
		DefaultLogger.Info(msg)
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Info(msg)
	}
}

func (p *Progress) Done() {
	if !p.pending {
		return
	}

	p.pending = false
	if p.jsonMode {
		return
	}
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
	if p.jsonMode {
		return
	}
	if p.writer != io.Discard {
		DefaultLogger.Error(p.lastMessage)
	}
}
