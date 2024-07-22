package testenv

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestService(t *testing.T) {
	Convey("Service", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Contract With RawMessage": contractWithRawInputOutput,
		}
		for title, fn := range testCases {
			Convey(title,
				fn(
					t, invokeEdgeServerFastHttp("edge", 8082, services.EchoRawService),
				),
			)
		}
	})
}

func contractWithRawInputOutput(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		// Set Key to instance 1
		resp := &services.EchoResponse{}
		err := stub.New("localhost:8082").REST().
			SetMethod("GET").
			SetPath("/echo").
			DefaultResponseHandler(
				func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

					return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
				},
			).
			Run(ctx).
			Error()
		c.So(err, ShouldBeNil)
		c.So(resp.X, ShouldEqual, "x")
		c.So(resp.XP, ShouldEqual, "xp")
		c.So(resp.Output, ShouldEqual, "output")
	}
}
