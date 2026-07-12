package filelog

import (
	"fmt"

	"github.com/clubpay/ronykit/x/rkit"
	log "github.com/clubpay/ronykit/x/telemetry/logkit"

	"gopkg.in/natefinch/lumberjack.v2"
)

type core struct {
	maxSize       int
	maxAge        int
	maxBackups    int
	fields        []log.Field
	sensitiveMask bool

	enc log.Encoder
	lvl log.Level
	ll  lumberjack.Logger
}

func New(filename string, opts ...Option) log.Core {
	c := &core{
		maxSize:    10,
		maxAge:     7,
		maxBackups: 0,
		lvl:        log.DebugLevel,
	}

	for _, o := range opts {
		o(c)
	}

	c.enc = log.NewEncoderBuilder().JsonEncoder()

	c.ll = lumberjack.Logger{
		Filename:   filename,
		MaxSize:    c.maxSize, // Megabytes
		MaxAge:     c.maxAge,  // days
		MaxBackups: c.maxBackups,
	}

	return c
}

func (c *core) Enabled(level log.Level) bool {
	return c.lvl.Enabled(level)
}

func (c *core) With(fields []log.Field) log.Core {
	c.fields = append(c.fields, fields...)

	return c
}

func (c *core) Check(entry log.Entry, ce *log.CheckedEntry) *log.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}

	return ce
}

func (c *core) Write(entry log.Entry, fields []log.Field) error {
	buf, err := c.enc.EncodeEntry(entry, fields)
	if err != nil {
		return err
	}

	_, err = c.ll.Write(
		rkit.S2B(
			fmt.Sprintf("%s:%6s:\t %s %s",
				entry.Time.Format("06/01/02 03:04:05PM"),
				entry.Level.CapitalString(),
				entry.Message,
				rkit.B2S(buf.Bytes()),
			),
		),
	)
	if err != nil {
		return err
	}

	buf.Free()

	return nil
}

func (c *core) Sync() error {
	return nil
}
