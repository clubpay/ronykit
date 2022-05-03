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

func (s *southBridge) OnJoin(members ...ClusterMember) {
	// Maybe later we can do something
}

func (s *southBridge) OnLeave(memberIDs ...string) {
	// Maybe later we can do something
}
