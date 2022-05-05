package ronykit

import (
	"sync"
)

type southBridge struct {
	ctxPool sync.Pool
	l       Logger
	c       Cluster
	eh      ErrHandler
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

func wrapWithForwarder(c Contract) Contract {
	if c.EdgeSelector() == nil {
		return c
	}

	memberSel := c.EdgeSelector()
	cw := &contractWrap{
		Contract: c,
		preH: []HandlerFunc{
			func(ctx *Context) {
				_, err := memberSel(newLimitedContext(ctx))
				if err != nil {
					// TODO:: return error ?!
					return
				}

				// TODO:: implement it
				ctx.StopExecution()
			},
		},
	}

	return cw
}
