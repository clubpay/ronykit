package cache

import (
	"testing"
	"time"
)

func TestCachePartitionAndTTL(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("new cache error: %v", err)
	}

	part := c.Partition("p1")
	if ok := part.Set("k1", "v1"); !ok {
		t.Fatalf("expected set to succeed")
	}

	if v, ok := c.Get("k1"); ok || v != nil {
		t.Fatalf("expected no value without prefix")
	}

	var got any
	var ok bool
	for i := 0; i < 10; i++ {
		got, ok = part.Get("k1")
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ok || got != "v1" {
		t.Fatalf("expected value from partition, got %v", got)
	}

	part.Purge("k1")
	if _, ok := part.Get("k1"); ok {
		t.Fatalf("expected value to be purged")
	}
}
