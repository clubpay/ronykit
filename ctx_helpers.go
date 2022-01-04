package ronykit

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
