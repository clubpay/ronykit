package fasthttp

import (
	"encoding/json"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

type message struct {
	embeddedMessage

	A   string     `json:"a"`
	B   int        `json:"b"`
	C   []byte     `json:"c"`
	D   []string   `json:"d"`
	Sub subMessage `json:"sub"`
}

type embeddedMessage struct {
	X string `json:"x"`
	Y int    `json:"y"`
}

type subMessage struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func BenchmarkDecoder(b *testing.B) {
	b1, _ := json.Marshal(&message{
		A: "a",
		B: 1,
		C: []byte("c"),
		D: []string{"d1", "d2"},
		Sub: subMessage{
			A: "a",
			B: 1,
		},
		embeddedMessage: embeddedMessage{
			X: "x",
			Y: 10,
		},
	})

	reqCtx := &fasthttp.RequestCtx{}
	d := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&message{}))
	for i := 0; i < b.N; i++ {
		msg, err := d(reqCtx, b1)
		if err != nil {
			b.Fatal(err)
		}
		if msg.(*message).A != "a" { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
		if msg.(*message).Sub.B != 1 { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
		if msg.(*message).X != "x" { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
		if msg.(*message).Y != 10 { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
	}
}

func TestDecoder(t *testing.T) {
	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&message{}))

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.SetUserValue("a", "valueA")
	reqCtx.SetUserValue("b", "1")
	reqCtx.SetUserValue("c", "valueC")
	reqCtx.SetUserValue("d", "valueD")
	reqCtx.SetUserValue("x", "valueX")
	reqCtx.SetUserValue("y", "2")

	m, err := dec(reqCtx, nil)
	if err != nil {
		t.Fatal(err)
	}
	mm, ok := m.(*message)
	if !ok {
		t.Fatal("invalid type")
	}
	if mm.A != "valueA" {
		t.Fatal("invalid value for A")
	}
	if mm.B != 1 {
		t.Fatal("invalid value for B")
	}
	if string(mm.C) != "valueC" {
		t.Fatal("invalid value for C")
	}
	//if len(mm.D) != 1 || mm.D[0] != "valueD" {
	//	t.Fatal("invalid value for D")
	//}
	if mm.X != "valueX" {
		t.Fatal("invalid value for X - ", mm.X)
	}
	if mm.Y != 2 {
		t.Fatal("invalid value for Y - ", mm.Y)
	}
}
