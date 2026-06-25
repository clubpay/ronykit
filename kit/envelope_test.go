package kit_test

import (
	"bytes"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelope(t *testing.T) {
	tc := newTestConn(1, "", false)
	ctx := kit.NewContext(nil)
	e := kit.NewEnvelope(ctx, tc, true)
	e.
		SetHdr("K1", "V1").
		SetHdr("K2", "V2").
		SetMsg(kit.RawMessage("Some Random Message"))

	assert.Equal(t, "V1", e.GetHdr("K1"))
	assert.Equal(t, "V2", e.GetHdr("K2"))
	assert.Equal(t, kit.RawMessage("Some Random Message"), e.GetMsg())
}

type testMessage struct {
	StringKey string   `json:"stringKey"`
	IntKey    int      `json:"intKey"`
	BoolKey   bool     `json:"boolKey"`
	ArrString []string `json:"arrString"`
	ArrInt    []int    `json:"arrInt"`
	ArrBool   []bool   `json:"arrBool"`
}

func TestEnvelopeEncodeDecode(t *testing.T) {
	m := testMessage{
		StringKey: "Some Random Message",
		IntKey:    1,
		BoolKey:   true,
		ArrString: []string{"Some Random Message"},
		ArrInt:    []int{1, 2, 3},
		ArrBool:   []bool{true, false},
	}
	const mJSON = `{"stringKey":"Some Random Message","intKey":1,"boolKey":true,"arrString":["Some Random Message"],"arrInt":[1,2,3],"arrBool":[true,false]}` //nolint:lll

	b, err := kit.MarshalMessage(m)
	require.NoError(t, err)
	assert.Equal(t, mJSON, string(b))

	w := bytes.NewBuffer(nil)
	require.NoError(t, kit.EncodeMessage(m, w))
	assert.Equal(t, mJSON, w.String()[:w.Len()-1])
}
