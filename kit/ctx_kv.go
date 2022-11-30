package kit

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

type Store interface {
	Get(key string) any
	Exists(key string) bool
	Set(key string, val any)
	Delete(key string)
	Scan(prefix string, cb func(key string) bool)
}

// LocalStore returns a local store which could be used to share some key-values between different
// requests. If you need to have a shared key-value store between different instances of your server
// then you need to use ClusterStore method.
func (ctx *Context) LocalStore() Store {
	return ctx.ls
}
