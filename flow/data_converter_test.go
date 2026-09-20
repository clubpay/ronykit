package flow

import (
	"bytes"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
)

func TestEncryptedDataConverterRoundTrip(t *testing.T) {
	dc := EncryptedDataConverter("0123456789abcdef")

	payload, err := dc.ToPayload("hello")
	if err != nil {
		t.Fatal(err)
	}

	var got string
	if err := dc.FromPayload(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestEncryptedPayloadCodecPassesThroughUnreadable(t *testing.T) {
	codec := EncryptedPayloadCodec("0123456789abcdef")
	src := []*commonpb.Payload{{
		Data:     []byte("not-encrypted"),
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
	}}

	out, err := codec.Decode(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out[0].GetData(), src[0].GetData()) {
		t.Fatalf("plaintext payload was not passed through, got %q", out[0].GetData())
	}
	if !bytes.Equal(out[0].GetMetadata()["encoding"], []byte("json/plain")) {
		t.Fatalf("metadata was not passed through, got %q", out[0].GetMetadata()["encoding"])
	}
}

func TestEncryptedPayloadCodecDecodesMixedBatch(t *testing.T) {
	codec := EncryptedPayloadCodec("0123456789abcdef")

	encrypted, err := codec.Encode([]*commonpb.Payload{{Data: []byte("secret")}})
	if err != nil {
		t.Fatal(err)
	}

	plain := &commonpb.Payload{Data: []byte("legacy-plaintext")}

	out, err := codec.Decode([]*commonpb.Payload{plain, encrypted[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out[0].GetData(), plain.GetData()) {
		t.Fatalf("plaintext payload = %q", out[0].GetData())
	}
	if !bytes.Equal(out[1].GetData(), []byte("secret")) {
		t.Fatalf("encrypted payload did not decrypt, got %q", out[1].GetData())
	}
}

func TestEncryptedPayloadCodecPassesThroughOnBadMetadata(t *testing.T) {
	codec := EncryptedPayloadCodec("0123456789abcdef")

	encrypted, err := codec.Encode([]*commonpb.Payload{{
		Data:     []byte("secret"),
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
	}})
	if err != nil {
		t.Fatal(err)
	}

	// Data is readable but metadata is not; the payload must stay intact rather than
	// come back half-decrypted.
	encrypted[0].Metadata["encoding"] = []byte("clear")

	out, err := codec.Decode(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out[0].GetData(), encrypted[0].GetData()) {
		t.Fatal("payload with unreadable metadata was partially decoded")
	}
}

func TestEncryptedPayloadCodecDoesNotLeavePlaintext(t *testing.T) {
	codec := EncryptedPayloadCodec("0123456789abcdef")
	src := []*commonpb.Payload{{
		Data:     []byte("secret"),
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
	}}

	out, err := codec.Encode(src)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(out[0].GetData(), src[0].GetData()) {
		t.Fatal("encoded payload data was not encrypted")
	}

	decoded, err := codec.Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded[0].GetData(), src[0].GetData()) {
		t.Fatalf("decoded data = %q", decoded[0].GetData())
	}
}
