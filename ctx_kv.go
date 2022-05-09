package ronykit

const (
	CtxServiceName = "__ServiceName__"
	CtxContractID  = "__ContractID__"
	CtxRoute       = "__Route__"
)

func (ctx *Context) Route() string {
	route, _ := ctx.Get(CtxRoute).(string)

	return route
}

func (ctx *Context) ServiceName() string {
	svcName, _ := ctx.Get(CtxServiceName).(string)

	return svcName
}

func (ctx *Context) ContractID() string {
	contractID, _ := ctx.Get(CtxContractID).(string)

	return contractID
}

func (ctx *Context) Set(key string, val interface{}) *Context {
	ctx.Lock()
	ctx.kv[key] = val
	ctx.Unlock()

	return ctx
}

func (ctx *Context) Get(key string) interface{} {
	ctx.Lock()
	v := ctx.kv[key]
	ctx.Unlock()

	return v
}

func (ctx *Context) Walk(f func(key string, val interface{}) bool) {
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
