package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/bytedance/sonic"
	"github.com/clubpay/ronykit/example/ex-02-rest/api"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

var _ kit.MessageCodec = (*fastCodec)(nil)

type fastCodec struct{}

func (f *fastCodec) Encode(m kit.Message, w io.Writer) error {
	return sonic.ConfigDefault.NewEncoder(w).Encode(m)
}

func (f *fastCodec) Decode(m kit.Message, r io.Reader) error {
	return sonic.ConfigDefault.NewDecoder(r).Decode(m)
}

func (f *fastCodec) Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func (f *fastCodec) Unmarshal(data []byte, v interface{}) error {
	return sonic.Unmarshal(data, v)
}

func main() {
	runtime.GOMAXPROCS(4)

	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	// In case we want to use a more performant codec, we can replace it with
	// our custom codec. In this case, we use the sonic codec.
	// However, this is optional, and the default goccy/go-json is good enough.
	kit.SetCustomCodec(&fastCodec{})

	// Create, start, and wait for the shutdown signal of the server.
	defer kit.NewServer(
		// kit.WithPrefork(),
		kit.WithErrorHandler(func(ctx *kit.Context, err error) {
			fmt.Println(err, string(debug.Stack()))
		}),
		kit.WithGateway(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
				fasthttp.WithServerName("RonyKIT Server"),
				fasthttp.WithCORS(fasthttp.CORSConfig{}),
				fasthttp.WithWebsocketEndpoint("/ws"),
				fasthttp.WithPredicateKey("cmd"),
			),
		),
		kit.WithServiceBuilder(
			api.SampleDesc.Desc(),
		),
	).
		Start(context.TODO()).
		PrintRoutesCompact(os.Stdout).
		Shutdown(context.TODO(), syscall.SIGHUP)
}
