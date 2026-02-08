package p2pcluster

import (
	"bytes"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/x/batch"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
)

func TestNewAppliesOptions(t *testing.T) {
	c := New(
		"demo",
		WithBroadcastInterval(2*time.Second),
		WithListenHost("127.0.0.1"),
		WithListenPort(1234),
		WithLogger(kit.NOPLogger{}),
	).(*cluster)

	if c.broadcastInterval != 2*time.Second {
		t.Fatalf("unexpected broadcast interval: %v", c.broadcastInterval)
	}
	if c.listenHost != "127.0.0.1" {
		t.Fatalf("unexpected listen host: %s", c.listenHost)
	}
	if c.listenPort != 1234 {
		t.Fatalf("unexpected listen port: %d", c.listenPort)
	}
}

func TestGenSortMessagesOrdersBySeqno(t *testing.T) {
	input := make(chan *pubsub.Message, 2)
	fn := genSortMessages(input)

	msgA := &pubsub.Message{Message: &pb.Message{Seqno: []byte("b")}}
	msgB := &pubsub.Message{Message: &pb.Message{Seqno: []byte("a")}}

	entries := []batch.Entry[*pubsub.Message, batch.NA]{
		batch.NewEntry[*pubsub.Message, batch.NA](msgA, nil),
		batch.NewEntry[*pubsub.Message, batch.NA](msgB, nil),
	}

	fn("tag", entries)

	first := <-input
	second := <-input
	if bytes.Compare(first.GetSeqno(), second.GetSeqno()) > 0 {
		t.Fatalf("messages not ordered by seqno: %q then %q", first.GetSeqno(), second.GetSeqno())
	}
}

func TestPublishErrors(t *testing.T) {
	c := New("demo").(*cluster)

	if err := c.Publish("missing", []byte("x")); err != kit.ErrClusterMemberNotFound {
		t.Fatalf("expected ErrClusterMemberNotFound, got: %v", err)
	}

	c.subs["peer1"] = utils.TimeUnix() - int64(c.broadcastInterval/time.Second)*3
	if err := c.Publish("peer1", []byte("x")); err != kit.ErrClusterMemberNotActive {
		t.Fatalf("expected ErrClusterMemberNotActive, got: %v", err)
	}
}
