package ronykit

import (
	"context"
	"net"
)

type Cluster interface {
	Start() error
	Shutdown() error
	Members() []ClusterMember
	MemberByID(string) ClusterMember
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

type ClusterMember interface {
	ServerID() string
	GatewayAddr() []net.Addr
	Dial(ctx context.Context) (net.Conn, error)
}

type ClusterDelegate interface {
	OnJoin(members ...ClusterMember)
	OnLeave(memberIDs ...string)
}
