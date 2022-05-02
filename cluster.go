package ronykit

import (
	"context"
	"net"
)

type Cluster interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Members(ctx context.Context) ([]ClusterMember, error)
	MemberByID(ctx context.Context, id string) (ClusterMember, error)
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

type ClusterMember interface {
	ServerID() string
	AdvertisedURL() []string
	Dial(ctx context.Context) (net.Conn, error)
}

type ClusterDelegate interface {
	OnError(err error)
	OnJoin(members ...ClusterMember)
	OnLeave(memberIDs ...string)
}

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
