package kit

import "github.com/clubpay/ronykit/kit/errors"

var (
	ErrClusterNotSet          = errors.New("cluster is not set")
	ErrClusterMemberNotFound  = errors.New("cluster member not found")
	ErrClusterMemberNotActive = errors.New("cluster member not active")
)

// ClusterStore returns a key-value store which is shared between different instances of the cluster.
//
// NOTE: If you don't set any Cluster for your EdgeServer, then this method will panic.
// NOTE: If the cluster doesn't support key-value store, then this method will return nil.
func (ctx *Context) ClusterStore() ClusterStore {
	if ctx.sb == nil {
		panic(ErrClusterNotSet)
	}

	s, ok := ctx.sb.cb.(ClusterWithStore)
	if ok {
		return s.Store()
	}

	return nil
}

// HasCluster returns true if the cluster is set for this EdgeServer.
func (ctx *Context) HasCluster() bool {
	return ctx.sb != nil
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
