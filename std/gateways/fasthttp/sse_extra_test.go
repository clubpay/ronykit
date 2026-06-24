package fasthttp

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
)

func TestSSESelector(t *testing.T) {
	sel := SSE("/stream")
	if sel.GetMethod() != MethodGet || sel.GetPath() != "/stream" {
		t.Fatalf("unexpected selector route: %s %s", sel.GetMethod(), sel.GetPath())
	}
	if !sel.IsStream() {
		t.Fatal("expected stream selector")
	}
	if sel.Query(queryStream) != true {
		t.Fatal("expected stream query flag")
	}

	post := SSEMethod(MethodPost, "/events")
	if post.GetMethod() != MethodPost || !post.IsStream() {
		t.Fatalf("unexpected post sse selector: %+v", post)
	}

	plain := GET("/plain")
	if plain.IsStream() {
		t.Fatal("expected non-stream GET selector")
	}
}

func TestSSEHTTPHandlerRESTStream(t *testing.T) {
	gw, _ := New()
	b := gw.(*bundle) //nolint:forcetypeassert

	delegate := &captureDelegate{msgCh: make(chan []byte, 1)}
	b.Subscribe(delegate)

	b.Register("svc", "c1", kit.JSON, SSE("/stream"), kit.RawMessage{}, kit.RawMessage{})

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = b.srv.Serve(ln)
	}()
	defer b.srv.Shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/stream", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != sseContentType {
		t.Fatalf("unexpected content type: %s", ct)
	}

	select {
	case <-delegate.msgCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive REST SSE message")
	}
}

func TestSSEHTTPConnStream(t *testing.T) {
	conn := &sseHTTPConn{}
	if !conn.Stream() {
		t.Fatal("expected streaming REST connection")
	}
}

func TestWriteSSEEvent(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	if err := writeSSEEvent(w, "message", []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "event: message\n") {
		t.Fatalf("unexpected event output: %q", got)
	}
	if !strings.Contains(got, `data: {"ok":true}`) {
		t.Fatalf("unexpected data output: %q", got)
	}
}

func TestGenHTTPHandlerUsesSSEForStreamRoutes(t *testing.T) {
	gw, _ := New()
	b := gw.(*bundle) //nolint:forcetypeassert

	streamHandler := b.genHTTPHandler(routeData{Stream: true})
	if streamHandler == nil {
		t.Fatal("expected stream handler")
	}

	plainHandler := b.genHTTPHandler(routeData{Stream: false})
	if plainHandler == nil {
		t.Fatal("expected plain handler")
	}
}
