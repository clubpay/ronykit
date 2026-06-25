package stub_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ipInfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Readme   string `json:"readme"`
	Timezone string `json:"timezone"`
}

func TestStubBasicFunctionality(t *testing.T) {
	ctx := context.Background()

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
					require.NoError(t, kit.UnmarshalMessage(r.GetBody(), v))
					assert.NotEmpty(t, v.Readme)
					assert.NotEmpty(t, v.IP)
				default:
					t.Skip("we got error from ipinfo.io")
				}

				return nil
			},
		).
		SetHeader("SomeKey", "SomeValue").
		Run(ctx)
	defer httpCtx.Release()

	assert.Nil(t, httpCtx.Err())
}

func TestStubFromURL(t *testing.T) {
	ctx := context.Background()

	httpCtx, err := stub.HTTP("https://ipinfo.io/json?someKey=someValue")
	require.NoError(t, err)
	httpCtx.
		SetMethod(http.MethodGet).
		DefaultResponseHandler(
			func(_ context.Context, r stub.RESTResponse) *stub.Error {
				switch r.StatusCode() {
				case http.StatusOK:
					v := &ipInfoResponse{}
					require.NoError(t, kit.UnmarshalMessage(r.GetBody(), v))
					assert.NotEmpty(t, v.Readme)
					assert.NotEmpty(t, v.IP)
				default:
					t.Skip("we got error from ipinfo.io")
				}

				return nil
			},
		).
		SetHeader("SomeKey", "SomeValue").
		Run(ctx)
	defer httpCtx.Release()

	assert.Nil(t, httpCtx.Err())
}

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

func TestStubAutoRun(t *testing.T) {
	ctx := context.Background()

	httpCtx, err := stub.HTTP("https://postman-echo.com")
	require.NoError(t, err)
	httpCtx.
		SetMethod(http.MethodGet).
		DefaultResponseHandler(
			func(_ context.Context, r stub.RESTResponse) *stub.Error {
				switch r.StatusCode() {
				case http.StatusOK:
					v := &postEchoResponse{}
					require.NoError(t, kit.UnmarshalMessage(r.GetBody(), v))
					assert.Equal(t, "someName", v.Args["name"])
					assert.Equal(t, "12345", v.Args["value"])
					assert.Equal(t, "someName", v.Args["namePtr"])
					assert.Equal(t, "12345", v.Args["valuePtr"])
					assert.Len(t, v.Args["strings"], 3)
					assert.Len(t, v.Args["ints"], 3)
					assert.Equal(t, "a", v.Args["strings"].([]any)[0])
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

	assert.Nil(t, httpCtx.Err())
}
