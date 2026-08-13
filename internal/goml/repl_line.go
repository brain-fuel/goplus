package goml

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// lineReader abstracts input so a terminal gets history and editing
// while pipes and tests take a plain buffered path.
type lineReader interface {
	ReadLine(prompt string) (string, error)
	// Raw reports whether the terminal is in raw mode, where the driver
	// no longer turns "\n" into "\r\n" and output must do it instead.
	Raw() bool
	Close()
}

// crlf translates bare newlines to CRLF for a raw-mode terminal. Without
// it every printed line leaves the cursor in the column it ended on, and
// output walks diagonally down the screen.
type crlf struct{ w io.Writer }

func (c crlf) Write(p []byte) (int, error) {
	var b []byte
	for i, ch := range p {
		if ch == '\n' && (i == 0 || p[i-1] != '\r') {
			b = append(b, '\r')
		}
		b = append(b, ch)
	}
	if _, err := c.w.Write(b); err != nil {
		return 0, err
	}
	return len(p), nil
}

func newLineReader(in io.Reader, out io.Writer, interactive bool) lineReader {
	if f, ok := in.(*os.File); ok && interactive {
		if t, err := newTermReader(f, out); err == nil {
			return t
		}
	}
	return &bufReader{sc: bufio.NewScanner(in), out: out, echo: interactive}
}

// bufReader reads whole lines with no editing; used for pipes, scripts,
// and tests, where raw mode would be wrong.
type bufReader struct {
	sc   *bufio.Scanner
	out  io.Writer
	echo bool
}

func (b *bufReader) ReadLine(prompt string) (string, error) {
	if b.echo {
		fmt.Fprint(b.out, prompt)
	}
	if !b.sc.Scan() {
		if err := b.sc.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return b.sc.Text(), nil
}

func (b *bufReader) Raw() bool { return false }

func (b *bufReader) Close() {}

// termReader gives arrow-key history and cursor editing on a terminal.
type termReader struct {
	fd    int
	state *term.State
	t     *term.Terminal
}

func newTermReader(f *os.File, out io.Writer) (*termReader, error) {
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("not a terminal")
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	rw := struct {
		io.Reader
		io.Writer
	}{f, out}
	return &termReader{fd: fd, state: state, t: term.NewTerminal(rw, "")}, nil
}

func (t *termReader) ReadLine(prompt string) (string, error) {
	t.t.SetPrompt(prompt)
	line, err := t.t.ReadLine()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r"), nil
}

func (t *termReader) Raw() bool { return true }

func (t *termReader) Close() {
	if t.state != nil {
		_ = term.Restore(t.fd, t.state)
	}
}

// isTerminal reports whether input is an interactive terminal, which
// decides prompting and line editing.
func isTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
