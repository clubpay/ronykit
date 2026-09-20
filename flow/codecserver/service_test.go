package codecserver

import (
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestServiceDescRoutes(t *testing.T) {
	svc := NewService("/temporal-codec", map[string]string{"*": "0123456789abcdef"})
	desc := svc.Desc()

	got := map[string]string{}
	for _, c := range desc.Contracts {
		if len(c.RouteSelectors) == 0 {
			t.Fatalf("contract %q has no routes", c.Name)
		}

		sel, ok := c.RouteSelectors[0].Selector.(fasthttp.Selector)
		if !ok {
			t.Fatalf("contract %q selector type %T", c.Name, c.RouteSelectors[0].Selector)
		}

		got[c.Name] = sel.GetMethod() + " " + sel.GetPath()
	}

	want := map[string]string{
		"Decode":     http.MethodPost + " /temporal-codec/decode",
		"Encode":     http.MethodPost + " /temporal-codec/encode",
		"DecodeCORS": http.MethodOptions + " /temporal-codec/decode",
		"EncodeCORS": http.MethodOptions + " /temporal-codec/encode",
	}
	for name, route := range want {
		if got[name] != route {
			t.Fatalf("contract %s: got %q want %q", name, got[name], route)
		}
	}
}

func TestDecodeUsesNamespaceHeader(t *testing.T) {
	key := "0123456789abcdef"
	svc := NewService("/temporal-codec", map[string]string{"prod": key})
	codec := flow.EncryptedPayloadCodec(key)

	payloads, err := converter.GetDefaultDataConverter().ToPayloads("hello")
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := codec.Encode(payloads.GetPayloads())
	if err != nil {
		t.Fatal(err)
	}

	body, err := protojson.Marshal(&commonpb.Payloads{Payloads: encoded})
	if err != nil {
		t.Fatal(err)
	}

	err = kit.NewTestContext().
		SetHandler(svc.Decode).
		Input(kit.RawMessage(body), kit.EnvelopeHdr{headerNamespace: "prod"}).
		Expect(func(e *kit.Envelope) error {
			if e.GetHdr(headerCORSAllowOrigin) != "*" {
				t.Fatalf("missing CORS allow origin, got %q", e.GetHdr(headerCORSAllowOrigin))
			}

			raw, ok := e.GetMsg().(kit.RawMessage)
			if !ok {
				t.Fatalf("unexpected msg %T", e.GetMsg())
			}

			var out commonpb.Payloads
			if err := protojson.Unmarshal(raw, &out); err != nil {
				return err
			}

			var got string
			if err := converter.GetDefaultDataConverter().FromPayloads(&out, &got); err != nil {
				return err
			}
			if got != "hello" {
				t.Fatalf("decoded %q", got)
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeUnknownNamespace(t *testing.T) {
	svc := NewService("/temporal-codec", map[string]string{"prod": "0123456789abcdef"})
	body, err := protojson.Marshal(&commonpb.Payloads{})
	if err != nil {
		t.Fatal(err)
	}

	err = kit.NewTestContext().
		SetHandler(svc.Decode).
		Input(kit.RawMessage(body), kit.EnvelopeHdr{headerNamespace: "other"}).
		Expect(func(e *kit.Envelope) error {
			if e.GetMsg() != "codec not found for namespace" {
				t.Fatalf("unexpected msg %#v", e.GetMsg())
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatal(err)
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	key := "0123456789abcdef"
	svc := NewService("/temporal-codec", map[string]string{"*": key})

	payloads, err := converter.GetDefaultDataConverter().ToPayloads("hello")
	if err != nil {
		t.Fatal(err)
	}

	body, err := protojson.Marshal(payloads)
	if err != nil {
		t.Fatal(err)
	}

	err = kit.NewTestContext().
		SetHandler(svc.Encode).
		Input(kit.RawMessage(body), nil).
		Expect(func(e *kit.Envelope) error {
			raw, ok := e.GetMsg().(kit.RawMessage)
			if !ok {
				t.Fatalf("unexpected msg %T", e.GetMsg())
			}

			var encoded commonpb.Payloads
			if err := protojson.Unmarshal(raw, &encoded); err != nil {
				return err
			}

			decoded, err := flow.EncryptedPayloadCodec(key).Decode(encoded.GetPayloads())
			if err != nil {
				return err
			}

			var got string
			if err := converter.GetDefaultDataConverter().FromPayloads(
				&commonpb.Payloads{Payloads: decoded},
				&got,
			); err != nil {
				return err
			}
			if got != "hello" {
				t.Fatalf("round-trip %q", got)
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCORSOptions(t *testing.T) {
	svc := NewService("/temporal-codec", map[string]string{"*": "0123456789abcdef"})

	err := kit.NewTestContext().
		SetHandler(svc.CORS).
		Input(kit.RawMessage{}, nil).
		Expect(func(e *kit.Envelope) error {
			if e.GetHdr(headerCORSAllowHeaders) != corsAllowHeaders {
				t.Fatalf("allow headers %q", e.GetHdr(headerCORSAllowHeaders))
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatal(err)
	}
}
