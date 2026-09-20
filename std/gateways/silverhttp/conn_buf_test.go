package silverhttp

import (
	"bytes"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/p"
	"github.com/clubpay/ronykit/x/rkit"
)

func TestWriteEnvelopeBufferContract(t *testing.T) {
	dataBuf := p.GetCap(64)
	defer dataBuf.Release()

	err := kit.EncodeMessage(kit.RawMessage(`{"ok":true}`), dataBuf)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}

	body := rkit.PtrVal(dataBuf.Bytes())
	if !bytes.Contains(body, []byte("ok")) {
		t.Fatalf("encoded body %q does not contain ok", body)
	}
	if dataBuf.Len() != len(body) {
		t.Fatalf("Len=%d, body=%d", dataBuf.Len(), len(body))
	}
}
