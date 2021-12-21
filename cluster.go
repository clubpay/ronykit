package ronykit

import (
	"context"
	"net"
)

type Cluster interface {
	Start()
	Shutdown()
	Members() []ClusterMember
	MemberByID(string) ClusterMember
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

type ClusterMember interface {
	ServerID() string
	GatewayAddr() []net.Addr
	TunnelAddr() []net.Addr
	Dial(ctx context.Context) (net.Conn, error)
}

type ClusterDelegate interface {
	OnJoin(hash uint64)
	OnLeave(hash uint64)
}
