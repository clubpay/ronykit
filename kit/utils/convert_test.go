package utils_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNumeric(t *testing.T) {
	tests := []struct {
		name string
		in   any
		xv   float64
		xs   string
	}{
		{"string", "13.14", 13.14, "13.14"},
		{"float64", 13.14, 13.14, "13.14"},
		{"float32", float32(13.14), 13.14, "13.14"},
		{"int", 13, 13.0, "13.00"},
		{"int64", int64(13), 13.0, "13.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			on := utils.ParseNumeric(tt.in)
			assert.InDelta(t, tt.xv, on.Value(), 1e-6)
			assert.Equal(t, tt.xs, on.String())
			b, err := json.Marshal(on)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("%q", tt.xs), string(b))
		})
	}
}

func TestNumericDecodedFromJSON(t *testing.T) {
	tests := []struct {
		name string
		str  string
		xv   float64
		xs   string
	}{
		{"string field", `{"f": "13.14"}`, 13.14, "13.14"},
		{"float field", `{"f": 13.14}`, 13.14, "13.14"},
		{"int field", `{"f": 13}`, 13.00, "13.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type medium struct {
				Field utils.Numeric `json:"f"`
			}
			m := new(medium)

			require.NoError(t, json.Unmarshal([]byte(tt.str), m))
			assert.InDelta(t, tt.xv, m.Field.Value(), 1e-6)
			assert.Equal(t, tt.xs, m.Field.String())
		})
	}
}

func TestNumericWithPrecision(t *testing.T) {
	tests := []struct {
		name string
		in   utils.Numeric
		prec int
		xv   float64
		xs   string
	}{
		{"increased", utils.ParseNumeric("13.14"), 3, 13.14, "13.140"},
		{"decreased", utils.ParseNumeric("13.14"), 1, 13.14, "13.1"},
		{"unchanged", utils.ParseNumeric("13.14"), 2, 13.14, "13.14"},
		{"unset", utils.ParseNumeric("13.14"), -1, 13.14, "13.14"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withPrec := tt.in.WithPrecision(tt.prec)
			assert.InDelta(t, tt.xv, withPrec.Value(), 1e-6)
			assert.Equal(t, tt.xs, withPrec.String())
		})
	}
}

func TestStrTruncate(t *testing.T) {
	tests := []struct {
		s       string
		maxSize int
		xs      string
	}{
		{"Merci Marcel Tiong Bahru", 13, "Merci Marcel "},
		{"Merci Marcel Tiong Bahru", 4, "Merc"},
		{"", 3, ""},
		{"Merci Marcel Tiong Bahru", 1, "M"},
		{"Merci Marcel Tiong Bahru", 0, ""},
		{"Merci Marcel Tiong Bahru", -1, ""},
		{" ", 0, ""},
		{" ", 1, " "},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q_%d", tt.s, tt.maxSize), func(t *testing.T) {
			assert.Equal(t, tt.xs, utils.StrTruncate(tt.s, tt.maxSize))
		})
	}
}
