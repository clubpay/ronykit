package ronykit

import (
	"context"
)

// Cluster identifies all instances of our EdgeServer. The implementation of the Cluster is not
// forced by the architecture. However, we can break down Cluster implementations into two categories:
// shared store, or gossip based clusters. In our std package, we provide a store based cluster, which
// could be integrated with other services with the help of implementing ClusterStore.
type Cluster interface {
	Bundle
	Dispatcher
	Members(ctx context.Context) ([]ClusterMember, error)
	MemberByID(ctx context.Context, id string) (ClusterMember, error)
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

// ClusterMember represents an EdgeServer instance in the Cluster.
type ClusterMember interface {
	ServerID() string
	AdvertisedURL() []string
}

// ClusterDelegate is the delegate that connects the Cluster to the rest of the system.
type ClusterDelegate interface {
	OnError(err error)
	OnJoin(members ...ClusterMember)
	OnLeave(memberIDs ...string)
	// OnMessage must be called whenever a new message arrives.
	OnMessage(c Conn, msg []byte)
}

// ClusterStore is the abstraction of the store for store based cluster.
type ClusterStore interface {
	// SetMember call whenever the node has changed its state or metadata.
	SetMember(ctx context.Context, clusterMember ClusterMember) error
	// GetMember returns the ClusterMember by its ID.
	GetMember(ctx context.Context, serverID string) (ClusterMember, error)
	// SetLastActive is called periodically and implementor must keep track of last active timestamps
	// of all members and return the correct set in GetActiveMembers.
	SetLastActive(ctx context.Context, serverID string, ts int64) error
	// GetActiveMembers must return a list of ClusterMembers that their last active is larger
	// than the lastActive input.
	GetActiveMembers(ctx context.Context, lastActive int64) ([]ClusterMember, error)
}

// southBridge is a container component that connects EdgeServer with a Cluster type Bundle.
type southBridge struct {
	ctxPool
	l  Logger
	c  Cluster
	eh ErrHandler
}

var _ ClusterDelegate = (*southBridge)(nil)

func (s *southBridge) OnError(err error) {
	s.eh(nil, err)
}

func (s *southBridge) OnJoin(members ...ClusterMember) {
	// Maybe later we can do something
}

func (s *southBridge) OnLeave(memberIDs ...string) {
	// Maybe later we can do something
}

func (s *southBridge) OnMessage(conn Conn, data []byte) {
	ctx := s.acquireCtx(conn)

	err := s.c.Dispatch(ctx, data, ctx.execute)
	if err != nil {
		s.eh(ctx, err)
	}

	s.releaseCtx(ctx)

	return
}

func requestForwarder(c Contract) Contract {
	if c.EdgeSelector() == nil {
		return c
	}

	memberSel := c.EdgeSelector()
	mw := func(ctx *Context) {
		_, err := memberSel(newLimitedContext(ctx))
		if err != nil {
			return
		}

		// 1. prepare a cluster message (contextData + input envelope)
		// 2. receives an array of [](contextDataSnapshot + output envelope)
		// 3. apply contextDataSnapshot and send the envelope.
		// ctx.StopExecution()
	}

	cw := &contractWrap{
		Contract: c,
		preH: []HandlerFunc{
			mw,
		},
	}

	return cw
}
