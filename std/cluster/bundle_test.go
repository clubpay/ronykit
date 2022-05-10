package cluster_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/std/cluster"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCluster(t *testing.T) {
	Convey("Cluster", t, func(c C) {
		store := &testStore{}
		clusters := make([]ronykit.Cluster, 0)
		for i := 0; i < 10; i++ {
			cl := cluster.MustNew(
				cluster.WithHeartbeat(time.Millisecond),
				cluster.WithID(fmt.Sprintf("Node%d", i)),
				cluster.WithStore(store),
			)

			err := cl.Start(context.Background())
			c.So(err, ShouldBeNil)

			clusters = append(clusters, cl)
		}

		time.Sleep(time.Second * 3)
		for _, cl := range clusters {
			members, err := cl.Members(context.Background())
			c.So(err, ShouldBeNil)
			c.So(members, ShouldHaveLength, 10)
		}

		for i := 0; i < 10; i++ {
			err := clusters[i].Shutdown(context.Background())
			c.So(err, ShouldBeNil)
		}
	})
}

type testStore struct {
	sync.Mutex
	members       map[string]ronykit.ClusterMember
	membersActive map[string]int64
}

var _ ronykit.ClusterStore = (*testStore)(nil)

func (t *testStore) SetMember(ctx context.Context, clusterMember ronykit.ClusterMember) error {
	t.Lock()
	defer t.Unlock()

	if t.members == nil {
		t.members = map[string]ronykit.ClusterMember{}
		t.membersActive = map[string]int64{}
	}

	t.members[clusterMember.ServerID()] = clusterMember

	return nil
}

func (t *testStore) GetMember(ctx context.Context, serverID string) (ronykit.ClusterMember, error) {
	t.Lock()
	defer t.Unlock()

	cm := t.members[serverID]
	if cm == nil {
		return nil, fmt.Errorf("not found")
	}

	return cm, nil
}

func (t *testStore) SetLastActive(ctx context.Context, serverID string, ts int64) error {
	t.Lock()
	defer t.Unlock()

	t.membersActive[serverID] = ts

	return nil
}

func (t *testStore) GetActiveMembers(ctx context.Context, lastActive int64) ([]ronykit.ClusterMember, error) {
	t.Lock()
	defer t.Unlock()

	members := make([]ronykit.ClusterMember, 0)
	for id, ts := range t.membersActive {
		if ts > lastActive {
			members = append(members, t.members[id])
		}
	}

	return members, nil
}
