package kit

import (
	"errors"
	"testing"
)

func TestTestContextRunRESTAndExpectations(t *testing.T) {
	ctx := NewTestContext().
		SetClientIP("127.0.0.1").
		Input(&inMessage{Name: "x"}, EnvelopeHdr{"k": "v"}).
		SetHandler(func(ctx *Context) {
			ctx.Out().SetMsg(&outMessage{OK: true}).Send()
		})

	err := ctx.
		Expect(func(e *Envelope) error {
			if e.GetHdr("missing") != "" {
				return errors.New("unexpected header")
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTestContextReceiverAndMismatch(t *testing.T) {
	ctx := NewTestContext().
		Input(&inMessage{Name: "x"}, EnvelopeHdr{}).
		SetHandler(func(ctx *Context) {
			ctx.Out().SetMsg(&outMessage{OK: true}).Send()
		})

	got := 0
	err := ctx.
		Receiver(func(out ...*Envelope) error {
			got = len(out)

			return nil
		}).
		Run(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1 {
		t.Fatalf("unexpected receiver output: %d", got)
	}

	ctx2 := NewTestContext().
		Input(&inMessage{Name: "x"}, EnvelopeHdr{}).
		SetHandler(func(ctx *Context) {})

	ctx2.Expect(func(*Envelope) error { return nil })
	if err := ctx2.RunREST(); err != ErrExpectationsDontMatch {
		t.Fatalf("expected ErrExpectationsDontMatch, got: %v", err)
	}
}

func TestTestContextExpectHelper(t *testing.T) {
	ctx := NewTestContext().
		SetHandler(func(ctx *Context) {
			ctx.Out().SetMsg(outMessage{OK: true}).Send()
		})

	var out outMessage
	var errMsg RawMessage
	if err := Expect(ctx, &inMessage{Name: "x"}, &out, &errMsg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.OK {
		t.Fatal("expected output to be set")
	}
}

func TestTestConnAndRESTConnMethods(t *testing.T) {
	conn := newTestConn()
	conn.clientIP = "1.2.3.4"
	conn.stream = true

	if conn.ConnID() == 0 {
		t.Fatal("expected conn id")
	}
	if conn.ClientIP() != "1.2.3.4" {
		t.Fatalf("unexpected client ip: %s", conn.ClientIP())
	}
	if _, err := conn.Write([]byte("x")); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if !conn.Stream() {
		t.Fatal("expected stream flag")
	}

	conn.Set("k", "v")
	if conn.Get("k") != "v" {
		t.Fatalf("unexpected Get value: %s", conn.Get("k"))
	}
	if len(conn.Keys()) != 1 {
		t.Fatalf("unexpected keys count: %d", len(conn.Keys()))
	}

	walked := false
	conn.Walk(func(key string, val string) bool {
		walked = true

		return false
	})
	if !walked {
		t.Fatal("expected Walk to be called")
	}

	rest := newTestRESTConn()
	rest.host = "example.com"
	rest.method = "GET"
	rest.path = "/path"
	rest.requestURI = "/path?x=1"
	rest.SetStatusCode(201)
	rest.Redirect(301, "/other")
	rest.WalkQueryParams(func(key string, val string) bool {
		return true
	})
	if rest.GetHost() != "example.com" || rest.GetMethod() != "GET" || rest.GetPath() != "/path" {
		t.Fatalf("unexpected rest values: %s %s %s", rest.GetHost(), rest.GetMethod(), rest.GetPath())
	}
	if rest.GetRequestURI() != "/path?x=1" {
		t.Fatalf("unexpected request uri: %s", rest.GetRequestURI())
	}
}
