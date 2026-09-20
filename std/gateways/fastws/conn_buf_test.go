package fastws

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/p"
)

type captureDelegate struct {
	msg []byte
}

func (d *captureDelegate) OnOpen(kit.Conn) {}

func (d *captureDelegate) OnClose(uint64) {}

func (d *captureDelegate) OnMessage(_ kit.Conn, msg []byte) {
	d.msg = append([]byte(nil), msg...)
}

func TestExecMessageUsesPooledBuffer(t *testing.T) {
	payload := []byte(`{"ok":true}`)
	buf := p.FromBytes(payload)
	d := &captureDelegate{}

	wsc := &wsConn{}
	wsc.execMessage(d, buf)

	if string(d.msg) != string(payload) {
		t.Fatalf("OnMessage got %q, want %q", d.msg, payload)
	}
}

func TestGetCapEncodeRoundTrip(t *testing.T) {
	dataBuf := p.GetCap(32)
	defer dataBuf.Release()

	err := kit.EncodeMessage(kit.RawMessage("ping"), dataBuf)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	if got := string(*dataBuf.Bytes()); got != "ping" {
		t.Fatalf("encoded body = %q, want %q", got, "ping")
	}
}
