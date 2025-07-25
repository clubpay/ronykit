package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"syscall"

	"github.com/clubpay/ronykit/flow/codecserver"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func main() {
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
				fasthttp.WithCORS(fasthttp.CORSConfig{
					AllowedHeaders:    []string{"X-Namespace", "Content-Type", "Authorization"},
					AllowedMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowedOrigins:    []string{"https://cloud.temporal.io"},
					ExposedHeaders:    nil,
					IgnoreEmptyOrigin: true,
				}),
				fasthttp.WithPredicateKey("cmd"),
			),
		),
		kit.WithServiceBuilder(
			codecserver.NewService("", map[string]string{
				"*":   "defaultKey",
				"ns1": "ns1Key",
			}).Desc(),
		),
	).
		Start(context.TODO()).
		PrintRoutesCompact(os.Stdout).
		Shutdown(context.TODO(), syscall.SIGHUP)

}
