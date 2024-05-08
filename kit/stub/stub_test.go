package stub_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/stub"
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
						Expect(kit.UnmarshalMessage(r.GetBody(), v)).To(Succeed())
						Expect(v.Readme).To(Not(BeEmpty()))
						Expect(v.IP).To(Not(BeEmpty()))
					default:
						Skip("we got error from ipinfo.io")
					}

					return nil
				},
			).
			SetHeader("SomeKey", "SomeValue").
			Run(ctx)
		defer httpCtx.Release()

		Expect(httpCtx.Err()).To(BeNil())
	})
})

var _ = Describe("Stub from URL", func() {
	ctx := context.Background()

	It("should unmarshal response from json", func() {
		httpCtx, err := stub.HTTP("https://ipinfo.io/json?someKey=someValue")
		Expect(err).To(BeNil())
		httpCtx.
			SetMethod(http.MethodGet).
			DefaultResponseHandler(
				func(_ context.Context, r stub.RESTResponse) *stub.Error {
					switch r.StatusCode() {
					case http.StatusOK:
						v := &ipInfoResponse{}
						Expect(kit.UnmarshalMessage(r.GetBody(), v)).To(Succeed())
						Expect(v.Readme).To(Not(BeEmpty()))
						Expect(v.IP).To(Not(BeEmpty()))
					default:
						Skip("we got error from ipinfo.io")
					}

					return nil
				},
			).
			SetHeader("SomeKey", "SomeValue").
			Run(ctx)
		defer httpCtx.Release()

		Expect(httpCtx.Err()).To(BeNil())
	})
})
