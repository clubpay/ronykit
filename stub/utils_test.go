package stub_test

import (
	"testing"

	"github.com/clubpay/ronykit/stub"
	"github.com/stretchr/testify/assert"
)

var keyValues = func(in string) string {
	switch in {
	case "p1":
		return "value1"
	case "p2":
		return "value2"
	case "p3":
		return "value3"
	}

	return in
}

func TestFillURLParams(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			"/some/{p1}/{p2}?something={p3}&boolean",
			"/some/value1/value2?something=value3&boolean",
		},
		{
			"/some/{p1}/{p2}?something={p3}&boolean",
			"/some/value1/value2?something=value3&boolean",
		},
		{
			"/some/{p1}{p2}/?something={p3}&boolean",
			"/some/value1value2/?something=value3&boolean",
		},
		{
			"/some/{p1}",
			"/some/value1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.out, stub.FillParams(tt.in, keyValues))
		})
	}
}
