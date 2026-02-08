package fasthttp

import (
	"net"
	"reflect"
	"unsafe"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

func setUnexportedField(ptr any, name string, value any) {
	v := reflect.ValueOf(ptr).Elem().FieldByName(name)
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func newTestEnvelope(ctx *kit.Context, conn kit.Conn) *kit.Envelope {
	env := &kit.Envelope{}
	setUnexportedField(env, "kv", map[string]string{})
	setUnexportedField(env, "ctx", ctx)
	setUnexportedField(env, "conn", conn)
	return env
}

func newTestContext(conn kit.Conn) *kit.Context {
	ctx := &kit.Context{}
	setUnexportedField(ctx, "conn", conn)
	setUnexportedField(ctx, "hdr", map[string]string{})
	setUnexportedField(ctx, "kv", map[string]any{})
	env := newTestEnvelope(ctx, conn)
	setUnexportedField(ctx, "in", env)
	return ctx
}

func newRequestCtx(method, uri string) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	ctx.Init(&ctx.Request, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, nil)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	return &ctx
}
