package reflector_test

import (
	"testing"

	"github.com/clubpay/ronykit/reflector"
)

type testMessage struct {
	X string
	Y int64
	z string
	M map[string]string
}

func (t testMessage) Marshal() ([]byte, error) {
	return nil, nil
}

func TestReflector(t *testing.T) {
	r := reflector.New()
	m := &testMessage{
		X: "xValue",
		Y: 10,
		z: "zValue",
		M: nil,
	}
	t.Log(r.Get(m, "X"))
}
