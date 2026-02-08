package stacktrace

import (
	"strings"
	"testing"

	"github.com/clubpay/ronykit/kit/utils/buf"
)

func TestTakeStacktraceIncludesFunction(t *testing.T) {
	trace := TakeStacktrace(0)
	if trace == "" {
		t.Fatal("expected stacktrace to be non-empty")
	}
	if !strings.Contains(trace, "TestTakeStacktraceIncludesFunction") {
		t.Fatalf("expected stacktrace to include function name, got %q", trace)
	}
}

func TestStackFormatterEmitsFrames(t *testing.T) {
	stack := captureStacktrace(0, stacktraceFull)
	defer stack.Free()

	if stack.Count() == 0 {
		t.Fatal("expected stacktrace to have frames")
	}

	buffer := buf.GetCap(256)
	defer buffer.Release()

	stackfmt := newStackFormatter(buffer)
	stackfmt.FormatStack(stack)

	out := string(*buffer.Bytes())
	if out == "" {
		t.Fatal("expected formatted stacktrace output")
	}
}
