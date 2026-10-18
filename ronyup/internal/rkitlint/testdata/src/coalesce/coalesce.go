package coalesce

import "time"

type config struct {
	Name    string
	Port    int
	Timeout time.Duration
}

const defaultName = "svc"

func defaults(c config, fallback string) {
	if c.Name == "" { // want `default value assigned with if: use rkit.Coalesce`
		c.Name = fallback
	}

	if c.Port == 0 { // want `default value assigned with if: use rkit.Coalesce`
		c.Port = 8080
	}
	if c.Port == 0 {
		c.Port = 9090
	}

	if c.Timeout == 0 { // want `use rkit.Coalesce`
		c.Timeout = 5 * time.Second
	}
}

func name(primary, secondary string) string {
	if primary != "" { // want `chain of non-zero checks returning the first value: use rkit.Coalesce`
		return primary
	}

	if secondary != "" {
		return secondary
	}

	return defaultName
}

// Not reported below.

var global string

func packageVar() {
	if global == "" {
		global = "x"
	}
}

func throughPointer(c *config) {
	if c.Name == "" { // c is shared: Coalesce would write it unconditionally
		c.Name = "x"
	}
}

type holder struct{ c *config }

func fallbackDerefs(name string, h holder) string {
	if name == "" {
		name = h.c.Name // evaluated eagerly by Coalesce, even when name is set
	}

	return name
}

func callFallback(c config, def func() string) {
	if c.Name == "" {
		c.Name = def()
	}
}

func otherTarget(c, d *config) {
	if c.Name == "" {
		d.Name = "x"
	}
}

func withElse(c config) {
	if c.Name == "" {
		c.Name = "x"
	} else {
		c.Name += "!"
	}
}

func sliceDefault(s, def []string) []string {
	if s == nil {
		s = def
	}

	return s
}

func ifaceDefault(err, def error) error {
	if err == nil {
		err = def
	}

	return err
}

func notLast(primary string) string {
	if primary != "" {
		return primary
	}

	println("side effect")

	return "y"
}
