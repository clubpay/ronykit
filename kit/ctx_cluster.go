package kit

import "github.com/clubpay/ronykit/kit/errors"

var ErrClusterNotSet = errors.New("cluster is not set")

func (ctx *Context) ClusterSet(key, value string) error {
	if ctx.sb == nil {
		return ErrClusterNotSet
	}

	return ctx.sb.cb.SetKey(key, value)
}

func (ctx *Context) ClusterUnset(key string) error {
	if ctx.sb == nil {
		return ErrClusterNotSet
	}

	return ctx.sb.cb.DeleteKey(key)
}

func (ctx *Context) ClusterGet(key string) (string, error) {
	if ctx.sb == nil {
		return "", ErrClusterNotSet
	}

	return ctx.sb.cb.GetKey(key)
}

func (ctx *Context) ClusterID() string {
	if ctx.sb == nil {
		return ""
	}

	return ctx.sb.id
}
