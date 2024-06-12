package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type GetRequest struct {
	Key string `json:"key"`
}

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var SimpleKeyValueService kit.ServiceDescriptor = desc.NewService("simpleKeyValueService").
	AddContract(
		desc.NewContract().
			SetCoordinator(keyValueCoordinator).
			SetInput(&SetRequest{}).
			SetOutput(&KeyValue{}).
			Selector(
				fasthttp.POST("/set-key"),
			).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					req, _ := ctx.In().GetMsg().(*SetRequest)
					err := sharedKV.Set(ctx, req.Key, ctx.ClusterID(), 0)
					if err != nil {
						ctx.SetStatusCode(500)

						return
					}

					ctx.LocalStore().Set(req.Key, req.Value)

					ctx.Out().
						SetMsg(&KeyValue{Key: req.Key, Value: req.Value}).
						Send()
				}),
		desc.NewContract().
			SetCoordinator(keyValueCoordinator).
			SetInput(&GetRequest{}).
			SetOutput(&KeyValue{}).
			Selector(fasthttp.GET("/get-key/{key}")).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					req, _ := ctx.In().GetMsg().(*GetRequest)
					value := ctx.LocalStore().Get(req.Key)
					if value == nil {
						ctx.SetStatusCode(http.StatusNotFound)

						return
					}
					ctx.Conn().Walk(func(key string, val string) bool {
						fmt.Println("Conn:", key, val)

						return true
					})

					ctx.Conn().Set("Conn-Hdr-Out", ctx.Conn().Get("Conn-Hdr-In"))

					ctx.Out().
						SetHdr("Envelope-Hdr-Out", ctx.In().GetHdr("Envelope-Hdr-In")).
						SetMsg(&KeyValue{Key: req.Key, Value: value.(string)}). //nolint:forcetypeassert
						Send()
				},
			),
	)

func keyValueCoordinator(ctx *kit.LimitedContext) (string, error) {
	var key string
	switch ctx.In().GetMsg().(type) {
	case *SetRequest:
		key = ctx.In().GetMsg().(*SetRequest).Key //nolint:forcetypeassert
	case *GetRequest:
		key = ctx.In().GetMsg().(*GetRequest).Key //nolint:forcetypeassert
	}

	return sharedKV.Get(ctx, key)
}

func contextMW(t time.Duration) kit.HandlerFunc {
	return func(ctx *kit.Context) {
		userCtx, cf := context.WithTimeout(context.Background(), t)
		ctx.SetUserContext(userCtx)
		ctx.Next()
		cf()
	}
}

type testClusterStore struct {
	mtx      utils.SpinLock
	sharedKV map[string]string
}

func (t *testClusterStore) Set(ctx *kit.Context, key, value string, ttl time.Duration) error {
	cs := ctx.ClusterStore()
	if cs != nil {
		return cs.Set(key, value, ttl)
	}

	t.mtx.Lock()
	t.sharedKV[key] = value
	t.mtx.Unlock()

	return nil
}

func (t *testClusterStore) Get(ctx *kit.LimitedContext, key string) (string, error) {
	cs := ctx.ClusterStore()
	if cs != nil {
		return cs.Get(key)
	}

	t.mtx.Lock()
	value, ok := t.sharedKV[key]
	t.mtx.Unlock()
	if !ok {
		return "", fmt.Errorf("key not found")
	}

	return value, nil
}

var sharedKV = testClusterStore{
	sharedKV: make(map[string]string),
}
