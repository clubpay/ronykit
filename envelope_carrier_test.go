package ronykit_test

import (
	"net/http"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/utils"
	"github.com/goccy/go-json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type someRandomRequest1 struct {
	Param1     string   `json:"param1"`
	Param2     int64    `json:"param2"`
	SliceParam []string `json:"sliceParam"`
}

var _ = Describe("EnvelopeCarrier", func() {
	testServiceDesc := desc.NewService("test").
		AddContract(
			desc.NewContract().
				SetInput(&someRandomRequest1{}).
				SetEncoding(ronykit.JSON).
				AddSelector(testRESTSelector{
					enc:    ronykit.JSON,
					method: http.MethodGet,
					path:   "/some/random/url",
				}).
				SetHandler(func(ctx *ronykit.Context) {}),
		)

	conn := newTestConn(utils.RandomUint64(0), "", false)
	ctx := ronykit.NewContext()
	ctx.SetConn(conn)
	ctx.Conn().Set("k1", "v1")
	ctx.Exec(
		ronykit.ExecuteArg{
			ServiceName: "testService",
			ContractID:  "contract1",
			Route:       "somePredicate",
		},
		testServiceDesc.Generate().Contracts()[0],
	)
	ctx.In().
		SetHdr("envH1", "envVal1").
		SetMsg(
			&someRandomRequest1{
				Param1:     "something1",
				Param2:     4343,
				SliceParam: []string{"p1", "p2", "anotherP"},
			},
		)

	envCarrier1 := ronykit.EnvelopeCarrierFromContext(ctx)
	It("Must create EnvelopeCarrier from Context", func() {
		Expect(envCarrier1.ServiceName).To(Equal("testService"))
		Expect(envCarrier1.Hdr).To(HaveKeyWithValue("envH1", "envVal1"))
		Expect(envCarrier1.ConnHdr).To(HaveKeyWithValue("k1", "v1"))
		Expect(envCarrier1.IsREST).To(BeFalse())
	})

	envCarrierData, err := json.Marshal(envCarrier1)
	It("Must marshal EnvelopeCarrier to JSON", func() {
		Expect(err).To(BeNil())
	})

	envCarrier2 := ronykit.EnvelopeCarrierFromData(envCarrierData)
	It("Must create EnvelopeCarrier from JSON", func() {
		Expect(envCarrier2.ServiceName).To(Equal("testService"))
		Expect(envCarrier2.Hdr).To(HaveKeyWithValue("envH1", "envVal1"))
		Expect(envCarrier2.ConnHdr).To(HaveKeyWithValue("k1", "v1"))
		Expect(envCarrier2.IsREST).To(BeFalse())
	})
})
