package testenv

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestKitWithCluster(t *testing.T) {
	Convey("Kit with Cluster", t, func(c C) {
		testCases := map[string]fx.Option{
			"KeyValue Store - With Redis": fx.Options(
				// fx.Invoke(invokeRedisMonitor),
				invokeEdgeServerWithRedis("edge1", 8082, services.SimpleKeyValueService),
				invokeEdgeServerWithRedis("edge2", 8083, services.SimpleKeyValueService),
			),
			//"KeyValue Store - With P2P": fx.Options(
			//	invokeEdgeServerWithP2P("edge1", 8082, services.SimpleKeyValueService),
			//	invokeEdgeServerWithP2P("edge2", 8083, services.SimpleKeyValueService),
			//),
		}
		for title, opts := range testCases {
			Convey(title, kitWithCluster(t, opts))
		}
	})
}

func kitWithCluster(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		time.Sleep(time.Second * 5)
		hosts := []string{"localhost:8082", "localhost:8083"}
		for range 100 {
			key := "K_" + utils.RandomID(10)
			value := "V_" + utils.RandomID(10)
			setHostIndex := utils.RandomInt(len(hosts))
			setHost := hosts[setHostIndex]
			getHost := hosts[(setHostIndex+1)%len(hosts)]
			// Set Key to instance 1
			resp := &services.KeyValue{}
			err := stub.New(setHost).REST().
				SetMethod("POST").
				DefaultResponseHandler(
					func(ctx context.Context, r stub.RESTResponse) *stub.Error {
						c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

						return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
					},
				).
				AutoRun(ctx, "/set-key", kit.JSON, &services.SetRequest{Key: key, Value: value}).
				Error()
			c.So(err, ShouldBeNil)
			c.So(resp.Key, ShouldEqual, key)
			c.So(resp.Value, ShouldEqual, value)

			// Get Key from instance 2
			connHdrIn := utils.RandomID(12)
			envelopeHdrIn := utils.RandomID(12)
			err = stub.New(getHost).REST().
				SetMethod("GET").
				SetHeader("Conn-Hdr-In", connHdrIn).
				SetHeader("Envelope-Hdr-In", envelopeHdrIn).
				DefaultResponseHandler(
					func(ctx context.Context, r stub.RESTResponse) *stub.Error {
						c.So(r.GetHeader("Conn-Hdr-Out"), ShouldEqual, connHdrIn)
						c.So(r.GetHeader("Envelope-Hdr-Out"), ShouldEqual, envelopeHdrIn)
						c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

						return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
					},
				).
				AutoRun(ctx, "/get-key/{key}", kit.JSON, &services.GetRequest{Key: key}).
				Error()
			c.So(err, ShouldBeNil)
			c.So(resp.Key, ShouldEqual, key)
			c.So(resp.Value, ShouldEqual, value)
		}
	}
}
