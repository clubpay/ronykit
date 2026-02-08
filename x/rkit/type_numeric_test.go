package rkit

import (
	"encoding/json"
	"testing"
)

func TestParseNumeric(t *testing.T) {
	if got := ParseNumeric(10).Value(); got != 10 {
		t.Fatalf("ParseNumeric int = %v, want 10", got)
	}
	if got := ParseNumeric(int64(11)).Value(); got != 11 {
		t.Fatalf("ParseNumeric int64 = %v, want 11", got)
	}
	if got := ParseNumeric(float64(1.25)).Value(); got != 1.25 {
		t.Fatalf("ParseNumeric float64 = %v, want 1.25", got)
	}
	if got := ParseNumeric(float32(1.5)).Value(); got != 1.5 {
		t.Fatalf("ParseNumeric float32 = %v, want 1.5", got)
	}
	if got := ParseNumeric("2.75").Value(); got != 2.75 {
		t.Fatalf("ParseNumeric string = %v, want 2.75", got)
	}
}

func TestNumericJSON(t *testing.T) {
	var n Numeric
	if err := json.Unmarshal([]byte(`"12.5"`), &n); err != nil {
		t.Fatalf("Unmarshal string = %v", err)
	}
	if n.Value() != 12.5 {
		t.Fatalf("Unmarshal string value = %v, want 12.5", n.Value())
	}

	if err := json.Unmarshal([]byte(`7`), &n); err != nil {
		t.Fatalf("Unmarshal number = %v", err)
	}
	if n.Value() != 7 {
		t.Fatalf("Unmarshal number value = %v, want 7", n.Value())
	}

	out, err := json.Marshal(ParseNumeric(2))
	if err != nil {
		t.Fatalf("Marshal = %v", err)
	}
	if string(out) != `"2.00"` {
		t.Fatalf("Marshal output = %q, want %q", string(out), `"2.00"`)
	}
}

func TestNumericPrecision(t *testing.T) {
	n := ParseNumeric(1.2345)
	if got := n.WithPrecision(3).String(); got != "1.234" {
		t.Fatalf("WithPrecision = %q, want %q", got, "1.234")
	}

	if got := n.WithoutPrecision().String(); got != "1.2345" {
		t.Fatalf("WithoutPrecision = %q, want %q", got, "1.2345")
	}
}
