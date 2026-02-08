package common

import (
	"io"
	"os"
	"strings"
	"testing"
)

type testPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestNopLoggerDoesNotPanic(t *testing.T) {
	logger := NewNopLogger()
	logger.Debugf("debug %d", 1)
	logger.Errorf("error %s", "boom")
}

func TestStdLoggerWritesToStdout(t *testing.T) {
	logger := NewStdLogger()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	logger.Debugf("hello %d", 1)
	logger.Errorf("boom %s", "error")

	_ = w.Close()
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()

	output := string(out)
	if !strings.Contains(output, "DEBUG: hello 1") {
		t.Fatalf("expected debug output, got %q", output)
	}
	if !strings.Contains(output, "ERROR: boom error") {
		t.Fatalf("expected error output, got %q", output)
	}
}

func TestSimpleRPCContainers(t *testing.T) {
	out := SimpleOutgoingJSONRPC().(*simpleOutgoingJSONRPC) //nolint:forcetypeassert
	out.SetID("req-1")
	out.SetHdr("x-trace", "trace-1")
	out.InjectMessage(&testPayload{Name: "alpha", Count: 2})

	data, err := out.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	in := SimpleIncomingJSONRPC().(*simpleIncomingJSONRPC) //nolint:forcetypeassert
	if err := in.Unmarshal(data); err != nil {
		t.Fatal(err)
	}

	if in.GetID() != "req-1" {
		t.Fatalf("expected id req-1, got %q", in.GetID())
	}
	if in.GetHdr("x-trace") != "trace-1" {
		t.Fatalf("expected header x-trace trace-1, got %q", in.GetHdr("x-trace"))
	}

	var payload testPayload
	if err := in.ExtractMessage(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Name != "alpha" || payload.Count != 2 {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	in.Release()
	out.Release()

	if in.ID != "" {
		t.Fatalf("expected released incoming id to be empty, got %q", in.ID)
	}
	if len(in.Header) != 0 {
		t.Fatalf("expected released incoming headers to be empty, got %v", in.Header)
	}
	if len(in.Payload) != 0 {
		t.Fatalf("expected released incoming payload to be empty, got %v", in.Payload)
	}
	if out.ID != "" {
		t.Fatalf("expected released outgoing id to be empty, got %q", out.ID)
	}
	if len(out.Header) != 0 {
		t.Fatalf("expected released outgoing headers to be empty, got %v", out.Header)
	}
	if out.Payload != nil {
		t.Fatalf("expected released outgoing payload to be nil, got %v", out.Payload)
	}
}
