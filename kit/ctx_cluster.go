package kit

import "github.com/clubpay/ronykit/kit/errors"

var ErrClusterNotSet = errors.New("cluster is not set")

// ClusterStore returns a key-value store which is shared between different instances of the cluster.
func (ctx *Context) ClusterStore() ClusterStore {
	if ctx.sb == nil {
		panic(ErrClusterNotSet)
	}

	return ctx.sb.cb.Store()
}

func (ctx *Context) ClusterID() string {
	if ctx.sb == nil {
		return ""
	}

	return ctx.sb.id
}

func (ctx *Context) ClusterMembers() ([]string, error) {
	if ctx.sb == nil {
		return nil, ErrClusterNotSet
	}

	return ctx.sb.cb.Subscribers()
}
