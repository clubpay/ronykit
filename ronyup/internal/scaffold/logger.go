package scaffold

import (
	"fmt"
	"strings"
)

// Logger receives progress messages during scaffolding.
type Logger interface {
	Printf(format string, a ...any)
	Println(a ...any)
	PrintErrf(format string, a ...any)
}

// DiscardLogger ignores all progress output.
type DiscardLogger struct{}

func (DiscardLogger) Printf(string, ...any)    {}
func (DiscardLogger) Println(...any)           {}
func (DiscardLogger) PrintErrf(string, ...any) {}

// BufferLogger collects progress output for MCP/tool responses.
type BufferLogger struct {
	b strings.Builder
}

func (l *BufferLogger) Printf(format string, a ...any) {
	fmt.Fprintf(&l.b, format, a...)
}

func (l *BufferLogger) Println(a ...any) {
	fmt.Fprintln(&l.b, a...)
}

func (l *BufferLogger) PrintErrf(format string, a ...any) {
	fmt.Fprintf(&l.b, format, a...)
}

func (l *BufferLogger) String() string { return l.b.String() }
