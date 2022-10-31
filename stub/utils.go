package stub

import (
	"strings"
	"unicode"

	"github.com/clubpay/ronykit"
)

// fillParams traverse the `pathPattern` and detect params with format `:name` and
// replace them using the function `f` provides in arguments.
func fillParams(pathPattern string, f func(key string) string) string {
	out := strings.Builder{}
	param := strings.Builder{}
	readingMode := false

	for _, r := range pathPattern {
		if readingMode {
			if !unicode.IsPunct(r) {
				param.WriteRune(r)

				continue
			}
			v := f(param.String())
			if v != "" {
				out.WriteString(v)
			} else {
				out.WriteString("_")
			}
			param.Reset()
			readingMode = false
		}
		if !readingMode && r == ':' {
			readingMode = true

			continue
		}

		out.WriteRune(r)
	}

	if param.Len() > 0 {
		v := f(param.String())
		if v != "" {
			out.WriteString(v)
		} else {
			out.WriteString("_")
		}
	}

	return out.String()
}

type nopLogger struct{}

var _ ronykit.Logger = nopLogger{}

func (n nopLogger) Debug(args ...interface{}) {}

func (n nopLogger) Debugf(format string, args ...interface{}) {}

func (n nopLogger) Error(args ...interface{}) {}

func (n nopLogger) Errorf(format string, args ...interface{}) {}
