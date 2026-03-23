package kit

import (
	"testing"
	"time"
)

func FuzzEnvelopeCarrierFromJSONNoPanic(f *testing.F) {
	f.Add([]byte(`{"id":"s","kind":1,"originID":"o","targetID":"t"}`))
	f.Add([]byte(`{"id":"s","kind":"bad"}`))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("FromJSON panicked: %v", r)
			}
		}()

		var ec envelopeCarrier
		_ = ec.FromJSON(data)
	})
}

func FuzzCastRawMessageNoPanic(f *testing.F) {
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`null`))
	f.Add([]byte("not-json"))

	f.Fuzz(func(t *testing.T, payload []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("CastRawMessage panicked: %v", r)
			}
		}()

		_, _ = CastRawMessage[map[string]any](RawMessage(payload))
	})
}

func FuzzFormatDurationAlwaysNonEmpty(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(time.Millisecond))
	f.Add(int64(-time.Second))

	f.Fuzz(func(t *testing.T, nanos int64) {
		out := formatDuration(time.Duration(nanos))
		if out == "" {
			t.Fatal("formatDuration returned empty output")
		}
	})
}
