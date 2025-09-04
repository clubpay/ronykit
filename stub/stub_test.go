package stub_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
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

type sampleRequest struct {
	Name     string   `json:"name"`
	NamePtr  *string  `json:"namePtr"`
	Value    int      `json:"value"`
	ValuePtr *int     `json:"valuePtr"`
	Strings  []string `json:"strings"`
	Ints     []int64  `json:"ints"`
}

type postEchoResponse struct {
	Args    map[string]any    `json:"args"`
	Headers map[string]string `json:"headers"`
	URL     string            `json:"url"`
}

var _ = Describe("Stub AutoRun", func() {
	ctx := context.Background()

	It("should handle pointer fields in the request", func() {
		httpCtx, err := stub.HTTP("https://postman-echo.com")
		Expect(err).To(BeNil())
		httpCtx.
			SetMethod(http.MethodGet).
			DefaultResponseHandler(
				func(_ context.Context, r stub.RESTResponse) *stub.Error {
					switch r.StatusCode() {
					case http.StatusOK:
						v := &postEchoResponse{}
						Expect(kit.UnmarshalMessage(r.GetBody(), v)).To(Succeed())
						Expect(v.Args["name"]).To(Equal("someName"))
						Expect(v.Args["value"]).To(Equal("12345"))
						Expect(v.Args["namePtr"]).To(Equal("someName"))
						Expect(v.Args["valuePtr"]).To(Equal("12345"))
						Expect(v.Args["strings"]).To(HaveLen(3))
						Expect(v.Args["ints"]).To(HaveLen(3))
						Expect(v.Args["strings"].([]any)[0]).To(Equal("a"))
					default:
						return stub.NewError(http.StatusInternalServerError, "unexpected status code")
					}

					return nil
				},
			).
			SetHeader("SomeKey", "SomeValue").
			AutoRun(ctx, "get", kit.JSON, sampleRequest{
				Name:     "someName",
				Value:    12345,
				NamePtr:  utils.ValPtr("someName"),
				ValuePtr: utils.ValPtr(12345),
				Strings:  []string{"a", "b", "c"},
				Ints:     []int64{1, 2, 3},
			})
		defer httpCtx.Release()

		Expect(httpCtx.Err()).To(BeNil())
	})
})
