package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Exporter writes human-readable multiline logs for terminal use.
type Exporter struct {
	w      io.Writer
	mu     sync.Mutex
	closed bool
	colors palette
}

type config struct {
	writer io.Writer
	color  *bool
}

// Option configures a terminal Exporter.
type Option func(*config)

// WithWriter sets the output writer. Defaults to os.Stdout.
func WithWriter(w io.Writer) Option {
	return func(cfg *config) {
		cfg.writer = w
	}
}

// WithColor forces color on or off. When unset, color is enabled for TTY writers.
func WithColor(enabled bool) Option {
	return func(cfg *config) {
		value := enabled
		cfg.color = &value
	}
}

// New creates a terminal-friendly log exporter.
func New(opts ...Option) *Exporter {
	cfg := config{
		writer: os.Stdout,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.writer == nil {
		cfg.writer = os.Stdout
	}

	return &Exporter{
		w:      cfg.writer,
		colors: newPalette(cfg.writer, cfg.color),
	}
}

// Export writes formatted log records to the configured writer.
func (e *Exporter) Export(ctx context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}

	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, err := fmt.Fprint(e.w, formatRecord(record, e.colors)); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(e.w); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown marks the exporter as closed.
func (e *Exporter) Shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.closed = true

	return nil
}

// ForceFlush performs no action.
func (e *Exporter) ForceFlush(context.Context) error {
	return nil
}
