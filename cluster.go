package ronykit

import (
	"context"
)

// EdgeSelector is a function that returns the target EdgeServer in the Cluster. If you have
// multiple instances of the EdgeServer, and you want to forward some requests to a specific
// instance, you can set up this function in desc.Contract, then, the receiving EdgeServer will
// detect the target instance by calling this function and forwards the request to the returned
// instance. From the external client point of view this forwarding request is not observable.
type EdgeSelector func(ctx *LimitedContext) (ClusterMember, error)

// Cluster identifies all instances of our EdgeServer. The implementation of the Cluster is not
// forced by the architecture. However, we can break down Cluster implementations into two categories:
// shared store, or gossip based clusters. In our std package, we provide a store based cluster, which
// could be integrated with other services with the help of ClusterStore.
type Cluster interface {
	Bundle
	Dispatcher
	Members(ctx context.Context) ([]ClusterMember, error)
	MemberByID(ctx context.Context, id string) (ClusterMember, error)
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

// ClusterMember represents an EdgerServer instance in the Cluster.
type ClusterMember interface {
	ServerID() string
	AdvertisedURL() []string
}

// ClusterDelegate is the delegate that connects the Cluster to the rest of the system.
type ClusterDelegate interface {
	OnError(err error)
	OnJoin(members ...ClusterMember)
	OnLeave(memberIDs ...string)
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
