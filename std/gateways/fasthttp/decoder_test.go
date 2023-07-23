package fasthttp

import (
	"encoding/json"
	"testing"

	"github.com/clubpay/ronykit/kit"
)

type message struct {
	A   string     `json:"a"`
	B   int        `json:"b"`
	C   []byte     `json:"c"`
	D   []string   `json:"d"`
	Sub subMessage `json:"sub"`
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
	})
	p := Params{}
	d := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&message{}))
	for i := 0; i < b.N; i++ {
		msg, err := d(p, b1)
		if err != nil {
			b.Fatal(err)
		}
		if msg.(*message).A != "a" { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
		if msg.(*message).Sub.B != 1 { //nolint:forcetypeassert
			b.Fatal("invalid value")
		}
	}
}
