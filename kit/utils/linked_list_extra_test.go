package utils_test

import (
	"strings"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestLinkedListOperations(t *testing.T) {
	ll := utils.NewLinkedList()
	if ll.Size() != 0 {
		t.Fatalf("expected empty list size, got %d", ll.Size())
	}

	ll.Append("a")
	ll.Append("b")
	ll.Prepend("c")

	if ll.Size() != 3 {
		t.Fatalf("unexpected list size: %d", ll.Size())
	}
	if ll.Head() == nil || ll.Head().GetData() != "c" {
		t.Fatalf("unexpected head node: %v", ll.Head())
	}
	if ll.Tail() == nil || ll.Tail().GetData() != "b" {
		t.Fatalf("unexpected tail node: %v", ll.Tail())
	}

	if got := ll.PickHeadData(); got != "c" {
		t.Fatalf("unexpected head data: %v", got)
	}
	if got := ll.PickTailData(); got != "b" {
		t.Fatalf("unexpected tail data: %v", got)
	}

	if ll.Size() != 1 {
		t.Fatalf("unexpected list size after picks: %d", ll.Size())
	}

	ll.Append("d")
	ll.Append("e")
	if n := ll.Get(1); n == nil || n.GetData() != "d" {
		t.Fatalf("unexpected node at index 1: %v", n)
	}

	ll.RemoveAt(1)
	if ll.Size() != 2 {
		t.Fatalf("unexpected list size after remove: %d", ll.Size())
	}

	out := ll.String()
	if !strings.Contains(out, "0.") || !strings.Contains(out, "1.") {
		t.Fatalf("unexpected list string: %q", out)
	}

	ll.Reset()
	if ll.Size() != 0 {
		t.Fatalf("expected list to be reset, got size %d", ll.Size())
	}
	if ll.PickHeadData() != nil {
		t.Fatal("expected nil after reset")
	}
}
