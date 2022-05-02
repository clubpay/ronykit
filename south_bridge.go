package ronykit

import "sync"

var _ ClusterDelegate = (*southBridge)(nil)

type southBridge struct {
	ctxPool sync.Pool
	l       Logger
	c       Cluster
	eh      ErrHandler
}

func (s *southBridge) OnError(err error) {
	s.eh(nil, err)
}

func (s *southBridge) OnJoin(members ...ClusterMember) {}

func (s *southBridge) OnLeave(memberIDs ...string) {}
