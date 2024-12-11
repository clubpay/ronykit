package kit_test

import (
	"bytes"

	"github.com/clubpay/ronykit/kit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Envelope", func() {
	tc := newTestConn(1, "", false)
	ctx := kit.NewContext(nil)
	e := kit.NewEnvelope(ctx, tc, true)
	e.
		SetHdr("K1", "V1").
		SetHdr("K2", "V2").
		SetMsg(kit.RawMessage("Some Random Message"))

	It("should read header and message", func() {
		Expect(e.GetHdr("K1")).To(Equal("V1"))
		Expect(e.GetHdr("K2")).To(Equal("V2"))
		Expect(e.GetMsg()).To(BeEquivalentTo("Some Random Message"))
	})
})

type testMessage struct {
	StringKey string   `json:"stringKey"`
	IntKey    int      `json:"intKey"`
	BoolKey   bool     `json:"boolKey"`
	ArrString []string `json:"arrString"`
	ArrInt    []int    `json:"arrInt"`
	ArrBool   []bool   `json:"arrBool"`
}

var _ = Describe("Envelope Encode / Decode", func() {
	m := testMessage{
		StringKey: "Some Random Message",
		IntKey:    1,
		BoolKey:   true,
		ArrString: []string{"Some Random Message"},
		ArrInt:    []int{1, 2, 3},
		ArrBool:   []bool{true, false},
	}
	const mJSON = `{"stringKey":"Some Random Message","intKey":1,"boolKey":true,"arrString":["Some Random Message"],"arrInt":[1,2,3],"arrBool":[true,false]}` //nolint:lll

	It("should encode and decode", func() {
		Expect(kit.MarshalMessage(m)).To(BeEquivalentTo(mJSON))
		w := bytes.NewBuffer(nil)
		Expect(kit.EncodeMessage(m, w)).To(BeNil())
		Expect(w.String()[:w.Len()-1]).To(BeEquivalentTo(mJSON))
	})
})
