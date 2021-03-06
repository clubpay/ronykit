package stub_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/stub"
	"github.com/goccy/go-json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stub Suite")
}

type ipInfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Readme   string `json:"readme"`
	Timezone string `json:"timezone"`
}

var _ = Describe("Stub Basic Functionality", func() {
	ctx := context.Background()

	It("should return 200 status", func() {
		s := stub.New("webhook.site", stub.Secure())
		httpCtx := s.REST().
			SetMethod(http.MethodGet).
			SetPath("/22fda9e7-1660-406e-b11e-993e070f175e").
			SetQuery("someKey", "someValue").
			Run(ctx)
		defer httpCtx.Release()

		Expect(httpCtx.Err()).To(BeNil())
		Expect(httpCtx.StatusCode()).To(Equal(http.StatusOK))
	})

	It("should unmarshal response from json", func() {
		s := stub.New("ipinfo.io", stub.Secure())
		httpCtx := s.REST().
			SetMethod(http.MethodGet).
			SetPath("/json").
			SetQuery("someKey", "someValue").
			DefaultResponseHandler(
				func(_ context.Context, r stub.RESTResponse) *stub.Error {
					switch r.StatusCode() {
					case http.StatusOK:
						v := &ipInfoResponse{}
						Expect(json.UnmarshalNoEscape(r.GetBody(), v)).To(Succeed())
						Expect(v.Readme).To(Not(BeEmpty()))
						Expect(v.IP).To(Not(BeEmpty()))
					default:
						Expect(r.StatusCode()).To(Equal(http.StatusOK))
					}

					return nil
				},
			).
			Run(ctx)
		defer httpCtx.Release()

		Expect(httpCtx.Err()).To(BeNil())
	})
})
