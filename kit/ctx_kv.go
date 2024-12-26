package kit

import "context"

var (
	kitCtxKey = struct{}{}
)

func (ctx *Context) setRoute(route string) *Context {
	ctx.route = append(ctx.route[:0], route...)

	return ctx
}

func (ctx *Context) Route() string {
	return string(ctx.route)
}

func (ctx *Context) setServiceName(s string) *Context {
	ctx.serviceName = append(ctx.serviceName[:0], s...)

	return ctx
}

func (ctx *Context) ServiceName() string {
	return string(ctx.serviceName)
}

func (ctx *Context) setContractID(contractID string) *Context {
	ctx.contractID = append(ctx.contractID[:0], contractID...)

	return ctx
}

func (ctx *Context) ContractID() string {
	return string(ctx.contractID)
}

// ContextWithValue is a helper function that helps us to update the kit.Context even if we don't
// have direct access to it. For example, we passed the ctx.Context() to one of the underlying
// functions in our code, and we need to set some key, which might be needed in one of our
// middlewares.
// In this case instead of:
//
//	ctx = context.WithValue(ctx, "myKey", "myValue")
//
// You can use this helper function:
//
//	ctx = kit.ContextWithValue(ctx, "myKey", "myValue")
//
// then later in your middleware you would access the key by simply doing:
//
//	ctx.Get("myKey")
func ContextWithValue(ctx context.Context, key string, value any) context.Context {
	kitCtx, ok := ctx.Value(kitCtxKey).(*Context)
	if !ok {
		return context.WithValue(ctx, key, value)
	}

	return kitCtx.Set(key, value).Context()
}

func (ctx *Context) Set(key string, val any) *Context {
	ctx.Lock()
	ctx.kv[key] = val
	ctx.ctx = context.WithValue(ctx.ctx, key, val)
	ctx.Unlock()

	return ctx
}

func (ctx *Context) Get(key string) any {
	ctx.Lock()
	v := ctx.kv[key]
	ctx.Unlock()

	return v
}

func (ctx *Context) Walk(f func(key string, val any) bool) {
	for k, v := range ctx.kv {
		if !f(k, v) {
			return
		}
	}
}

func (ctx *Context) Exists(key string) bool {
	return ctx.Get(key) != nil
}

func (ctx *Context) GetInt64(key string, defaultValue int64) int64 {
	v, ok := ctx.Get(key).(int64)
	if !ok {
		return defaultValue
	}

	return v
}

func (ctx *Context) GetInt32(key string, defaultValue int32) int32 {
	v, ok := ctx.Get(key).(int32)
	if !ok {
		return defaultValue
	}

	return v
}

func (ctx *Context) GetUint64(key string, defaultValue uint64) uint64 {
	v, ok := ctx.Get(key).(uint64)
	if !ok {
		return defaultValue
	}

	return v
}

func (ctx *Context) GetUint32(key string, defaultValue uint32) uint32 {
	v, ok := ctx.Get(key).(uint32)
	if !ok {
		return defaultValue
	}

	return v
}

func (ctx *Context) GetString(key string, defaultValue string) string {
	v, ok := ctx.Get(key).(string)
	if !ok {
		return defaultValue
	}

	return v
}

func (ctx *Context) GetBytes(key string, defaultValue []byte) []byte {
	v, ok := ctx.Get(key).([]byte)
	if !ok {
		return defaultValue
	}

	return v
}

// LocalStore returns a local store which could be used to share some key-values between different
// requests. If you need to have a shared key-value store between different instances of your server
// then you need to use ClusterStore method.
func (ctx *Context) LocalStore() Store {
	return ctx.ls
}
