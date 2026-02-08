package rediscluster

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewDefaultsAndOptions(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	cRaw, err := New(
		"demo",
		WithPrefix("custom"),
		WithGCPeriod(5*time.Second),
		WithRedisClient(client),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, ok := cRaw.(*cluster)
	if !ok {
		t.Fatalf("unexpected cluster type: %T", cRaw)
	}
	if c.prefix != "custom" {
		t.Fatalf("unexpected prefix: %s", c.prefix)
	}
	if c.gcPeriod != 5*time.Second {
		t.Fatalf("unexpected gc period: %v", c.gcPeriod)
	}
	if c.rc != client {
		t.Fatal("expected redis client to be set")
	}

	cRawDefault, err := New("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cDefault := cRawDefault.(*cluster)
	if cDefault.idleTime != 10*time.Minute {
		t.Fatalf("unexpected idle time: %v", cDefault.idleTime)
	}
	if cDefault.gcPeriod != time.Minute {
		t.Fatalf("unexpected gc period: %v", cDefault.gcPeriod)
	}
}

func TestWithRedisURLPanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid redis URL")
		}
	}()

	opt := WithRedisURL("://bad")
	c := &cluster{}
	opt(c)
}
