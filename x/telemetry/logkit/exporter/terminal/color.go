package terminal

import (
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type palette struct {
	enabled bool
}

func newPalette(w io.Writer, force *bool) palette {
	var enabled bool
	if force != nil {
		enabled = *force
	} else {
		enabled = isTerminalWriter(w)
	}

	return palette{enabled: enabled}
}

func isTerminalWriter(w io.Writer) bool {
	switch f := w.(type) {
	case *os.File:
		return term.IsTerminal(int(f.Fd()))
	default:
		return false
	}
}

func (p palette) reset() string {
	if !p.enabled {
		return ""
	}

	return "\033[0m"
}

func (p palette) paint(code, s string) string {
	if !p.enabled || s == "" {
		return s
	}

	return "\033[" + code + "m" + s + p.reset()
}

func (p palette) dim(s string) string {
	return p.paint("2", s)
}

func (p palette) bold(s string) string {
	return p.paint("1", s)
}

func (p palette) time(s string) string {
	return p.dim(s)
}

func (p palette) level(s string) string {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return p.paint("36", s)
	case "INFO":
		return p.paint("32", s)
	case "WARN", "WARNING":
		return p.paint("33", s)
	case "ERROR":
		return p.paint("31", s)
	case "DPANIC", "PANIC", "FATAL":
		return p.paint("1;31", s)
	default:
		return p.bold(s)
	}
}

func (p palette) traceID(s string) string {
	return p.paint("35", s)
}

func (p palette) location(s string) string {
	return p.paint("36", s)
}

func (p palette) body(s string) string {
	return p.bold(s)
}

func (p palette) attr(s string) string {
	return p.paint("90", s)
}
