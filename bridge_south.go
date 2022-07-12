package ronykit

import (
	"context"

	"github.com/goccy/go-json"
)

// Cluster identifies all instances of our EdgeServer. The implementation of the Cluster is not
// forced by the architecture. However, we can break down Cluster implementations into two categories:
// shared store, or gossip based clusters. In our std package, we provide a store based cluster, which
// could be integrated with other services with the help of implementing ClusterStore.
type Cluster interface {
	Bundle
	Members(ctx context.Context) ([]ClusterMember, error)
	MemberByID(ctx context.Context, id string) (ClusterMember, error)
	Me() ClusterMember
	Subscribe(d ClusterDelegate)
}

// ClusterMember represents an EdgeServer instance in the Cluster.
type ClusterMember interface {
	ServerID() string
	AdvertisedURL() []string
	RoundTrip(ctx context.Context, sendData []byte) (receivedData []byte, err error)
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

type ClusterChannel interface {
	RoundTrip(ctx context.Context, targetID string, sendData []byte) (receivedData []byte, err error)
}

// southBridge is a container component that connects EdgeServer with a Cluster type Bundle.
type southBridge struct {
	ctxPool
	l  Logger
	eh ErrHandler
	cr contractResolver
	c  Cluster
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

func (s *southBridge) OnMessage(conn Conn, data []byte) {}

func wrapWithCoordinator(c Contract) Contract {
	if c.EdgeSelector() == nil {
		return c
	}

	cw := &contractWrap{
		Contract: c,
		preH: []HandlerFunc{
			genForwarderHandler(c.EdgeSelector()),
		},
	}

	return cw
}

func genForwarderHandler(sel EdgeSelectorFunc) HandlerFunc {
	return func(ctx *Context) {
		target, err := sel(ctx.Limited())
		if err != nil {
			ctx.Error(err)

			return
		}

		out := envelopeCarrierFromContext(ctx)
		outData, err := json.Marshal(out)
		if err != nil {
			ctx.Error(err)

			return
		}

		inData, err := target.RoundTrip(ctx.Context(), outData)
		if err != nil {
			ctx.Error(err)

			return
		}

		in := envelopeCarrierFromData(inData)
		ctx.Out().
			SetHdrMap(in.Hdr).
			SetMsg(RawMessage(in.Msg)).
			Send()
	}
}
