// Package ptyrun runs an external command attached to a pseudo-terminal,
// streaming its output to a caller-provided writer. It has no knowledge of
// what the command is or how its output should be displayed — callers
// decide that by choosing what io.Writer and Sizer to pass in.
package ptyrun

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

// Sizer reports the desired pseudo-terminal size, in terminal cells.
// Implementations decide their own sizing policy; ptyrun just asks for a
// size before starting the command and again on every SIGWINCH.
type Sizer interface {
	Size() (cols, rows int)
}

// Run starts cmd attached to a new pseudo-terminal sized according to sz,
// and copies everything the command writes to out until the command
// exits.
//
// cmd must not have Stdin, Stdout, or Stderr already set; Run assigns
// them to the pseudo-terminal. If cmd was built with exec.CommandContext,
// cancelling that context kills the command exactly as it would for any
// other command in this codebase — Run does not add its own cancellation
// handling.
func Run(cmd *exec.Cmd, out io.Writer, sz Sizer) error {
	cols, rows := sz.Size()
	//nolint:gosec // cols/rows are clamped to sane positive ranges by the Sizer implementation
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}
	defer func() { _ = master.Close() }()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-winch:
				c, r := sz.Size()
				//nolint:gosec // cols/rows are clamped to sane positive ranges by the Sizer implementation
				_ = pty.Setsize(master, &pty.Winsize{Cols: uint16(c), Rows: uint16(r)})
			case <-stop:
				return
			}
		}
	}()

	copyDone := make(chan struct{})
	go func() {
		// Reading from a pty master after the slave side closes returns
		// an error (typically EIO on Linux) as its normal end-of-output
		// signal, not a real failure — it's discarded here on purpose.
		_, _ = io.Copy(out, master)
		close(copyDone)
	}()

	waitErr := cmd.Wait()
	close(stop)
	<-copyDone

	return waitErr
}
