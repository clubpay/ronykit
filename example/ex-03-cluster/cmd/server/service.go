package main

import (
	"fmt"
	"hash/crc32"
	"net/http"
	"sync"

	"github.com/clubpay/ronykit/example/ex-03-cluster/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

var (
	kvl sync.RWMutex
	kv  = map[string]string{}
	r   = reflector.New()
)

var serviceDesc desc.ServiceDescFunc = func() *desc.Service {
	return desc.NewService("SampleService").
		SetEncoding(kit.JSON).
		AddContract(
			desc.NewContract().
				SetCoordinator(coordinator).
				SetInput(&dto.SetKeyRequest{}).
				SetOutput(&dto.SetKeyResponse{}).
				AddRoute(desc.Route("", fasthttp.POST("/set"))).
				SetHandler(SetKeyHandler),
			desc.NewContract().
				SetCoordinator(coordinator).
				SetInput(&dto.GetKeyRequest{}).
				SetOutput(dto.Key{}).
				AddRoute(desc.Route("", fasthttp.GET("/get/:key"))).
				SetHandler(GetKeyHandler),
		)
}

func coordinator(ctx *kit.LimitedContext) (string, error) {
	members, err := ctx.ClusterMembers()
	if err != nil {
		return "", err
	}

	key, err := r.GetString(ctx.In().GetMsg(), "Key")
	if err != nil {
		return "", err
	}

	fmt.Println(
		"node", ctx.ClusterID(),
		"target", members[crc32.ChecksumIEEE(utils.S2B(key))%uint32(len(members))],
	)

	return members[crc32.ChecksumIEEE(utils.S2B(key))%uint32(len(members))], nil
}

func SetKeyHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.SetKeyRequest)
	kvl.Lock()
	kv[req.Key] = req.Value
	kvl.Unlock()

	ctx.In().Reply().
		SetHdr("ClusterID", ctx.ClusterID()).
		SetMsg(
			&dto.SetKeyResponse{
				Success: true,
			},
		).Send()
}

func GetKeyHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.GetKeyRequest)
	kvl.Lock()
	v := kv[req.Key]
	kvl.Unlock()

	ctx.SetStatusCode(http.StatusBadRequest)
	ctx.In().Reply().
		SetHdr("ClusterID", ctx.ClusterID()).
		SetMsg(
			&dto.Key{
				Value: v,
			},
		).Send()
	ctx.StopExecution()
}
