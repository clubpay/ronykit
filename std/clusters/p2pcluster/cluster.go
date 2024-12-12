package p2pcluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/batch"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type cluster struct {
	name       string
	id         string
	listenHost string
	listenPort int

	host      host.Host
	ps        *pubsub.PubSub
	discovery mdns.Service
	log       kit.Logger
	d         kit.ClusterDelegate

	gossipCancelFunc    context.CancelFunc
	broadcastCancelFunc context.CancelFunc
	broadcastInterval   time.Duration
	subsMtx             utils.SpinLock
	subs                map[string]int64
	subsList            []string
	topicsMtx           utils.SpinLock
	topics              map[string]*pubsub.Topic
	myTopicCancelFunc   context.CancelFunc
}

var (
	_ kit.Cluster  = (*cluster)(nil)
	_ mdns.Notifee = (*cluster)(nil)
)

func New(name string, opts ...Option) kit.Cluster {
	c := &cluster{
		name:              name,
		broadcastInterval: time.Minute,
		listenHost:        "0.0.0.0",
		listenPort:        0,
		log:               kit.NOPLogger{},
		subs:              make(map[string]int64),
		topics:            make(map[string]*pubsub.Topic),
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

func (c *cluster) Start(ctx context.Context) error {
	var err error
	c.host, err = libp2p.New(
		libp2p.Ping(true),
		libp2p.EnableRelay(),
		libp2p.EnableNATService(),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", c.listenHost, c.listenPort)),
	)
	if err != nil {
		return err
	}

	gossipCtx, gossipCF := context.WithCancel(context.Background())
	c.gossipCancelFunc = gossipCF
	c.ps, err = pubsub.NewGossipSub(gossipCtx, c.host)
	if err != nil {
		return err
	}

	// setup mDNS discovery to find local peers
	c.discovery = mdns.NewMdnsService(c.host, c.name, c)

	err = c.discovery.Start()
	if err != nil {
		return err
	}

	// Start my topic
	myTopicCtx, myTopicCF := context.WithCancel(context.Background())
	c.myTopicCancelFunc = myTopicCF
	err = c.startMyTopic(myTopicCtx)
	if err != nil {
		return err
	}

	// Start broadcast
	broadcastCtx, broadcastCF := context.WithCancel(context.Background())
	c.broadcastCancelFunc = broadcastCF
	err = c.startBroadcast(broadcastCtx)
	if err != nil {
		return err
	}

	return nil
}

func (c *cluster) startBroadcast(ctx context.Context) error {
	broadcastTopic, err := c.ps.Join(
		fmt.Sprintf("%s-broadcast", c.name),
	)
	if err != nil {
		return err
	}

	broadcastSub, err := broadcastTopic.Subscribe()
	if err != nil {
		return err
	}

	go func() {
		for {
			err = broadcastTopic.Publish(ctx, []byte(c.id))
			if err != nil {
				c.log.Errorf("[Cluster][p2pCluster] failed to broadcast: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.NewTicker(c.broadcastInterval).C:
			}
		}
	}()

	go func() {
		for {
			msg, err := broadcastSub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					c.log.Debugf("[p2pCluster] broadcast stopped")

					return
				}

				c.log.Errorf("[Cluster][p2pCluster] failed to receive broadcast message: %v", err)
				time.Sleep(time.Second)

				continue
			}

			if msg.ReceivedFrom == c.host.ID() {
				continue
			}

			c.subsMtx.Lock()
			c.subs[string(msg.GetData())] = utils.TimeUnix()
			c.subsList = utils.AddUnique(c.subsList, string(msg.GetData()))
			c.subsMtx.Unlock()
		}
	}()

	return nil
}

func (c *cluster) startMyTopic(ctx context.Context) error {
	topic, err := c.ps.Join(fmt.Sprintf("%s-%s", c.name, c.id))
	if err != nil {
		return err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	inputChan := make(chan *pubsub.Message, 1024)
	batcher := batch.NewMulti(
		genSortMessages(inputChan),
		batch.WithMaxWorkers(1),
		batch.WithMinWaitTime(time.Millisecond*25),
	)

	go c.receiveMessage(ctx, sub, batcher)
	go c.handleMessage(ctx, inputChan)

	return nil
}

func (c *cluster) receiveMessage(
	ctx context.Context, sub *pubsub.Subscription,
	b *batch.MultiBatcher[*pubsub.Message, batch.NA],
) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			c.log.Errorf("[Cluster][p2pCluster] failed to receive message from topic[%s]: %v", sub.Topic(), err)

			continue
		}

		if msg.ReceivedFrom == c.host.ID() {
			continue
		}

		b.Enter(
			msg.GetFrom().String(),
			batch.NewEntry[*pubsub.Message, batch.NA](msg, nil),
		)
	}
}

func (c *cluster) handleMessage(ctx context.Context, inputChan chan *pubsub.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-inputChan:
			c.d.OnMessage(msg.GetData())
		}
	}
}

func genSortMessages(
	inputChan chan *pubsub.Message,
) func(tagID string, entries []batch.Entry[*pubsub.Message, batch.NA]) {
	return func(tagID string, entries []batch.Entry[*pubsub.Message, batch.NA]) {
		sort.Slice(
			entries, func(i, j int) bool {
				return bytes.Compare(entries[i].Value().GetSeqno(), entries[j].Value().GetSeqno()) < 1
			},
		)

		for _, entry := range entries {
			inputChan <- entry.Value()
		}
	}
}

func (c *cluster) getTopic(id string) *pubsub.Topic {
	c.topicsMtx.Lock()
	defer c.topicsMtx.Unlock()

	t, ok := c.topics[id]
	if ok {
		return t
	}

	t, _ = c.ps.Join(fmt.Sprintf("%s-%s", c.name, id))
	c.topics[id] = t

	return t
}

func (c *cluster) Shutdown(ctx context.Context) error {
	c.broadcastCancelFunc()
	c.myTopicCancelFunc()
	c.gossipCancelFunc()

	_ = c.discovery.Close()
	_ = c.host.Close()

	return nil
}

func (c *cluster) Subscribe(id string, d kit.ClusterDelegate) {
	c.id = id
	c.d = d
}

func (c *cluster) Publish(id string, data []byte) error {
	c.subsMtx.Lock()
	t, ok := c.subs[id]
	c.subsMtx.Unlock()
	if !ok {
		return kit.ErrClusterMemberNotFound
	}

	if utils.TimeUnix()-t > int64(c.broadcastInterval/time.Second)*2 {
		return kit.ErrClusterMemberNotActive
	}

	return c.getTopic(id).Publish(context.Background(), data)
}

func (c *cluster) Subscribers() ([]string, error) {
	return c.subsList, nil
}

func (c *cluster) HandlePeerFound(pi peer.AddrInfo) {
	c.log.Debugf("[Cluster][p2pCluster] found peer: %s", pi.String())
	err := c.host.Connect(context.Background(), pi)
	if err != nil {
		c.log.Errorf("[Cluster][p2pCluster] failed to connect to peer(%s): %v", pi.String(), err)
	}
}
