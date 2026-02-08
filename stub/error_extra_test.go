package stub

import (
	"errors"
	"testing"
)

type testMsg struct {
	Code int
	Item string
}

func (t testMsg) GetCode() int { return t.Code }
func (t testMsg) GetItem() string {
	return t.Item
}

func TestErrorHelpers(t *testing.T) {
	err := NewError(400, "bad")
	if err.Code() != 400 || err.Item() != "bad" {
		t.Fatalf("unexpected error values: %d %s", err.Code(), err.Item())
	}
	if err.Error() != "ERR(400): bad" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}

	msg := testMsg{Code: 401, Item: "unauth"}
	err = NewErrorWithMsg(msg)
	if err.Code() != 401 || err.Item() != "unauth" {
		t.Fatalf("unexpected msg error values: %d %s", err.Code(), err.Item())
	}
	if err.GetCode() != 401 || err.GetItem() != "unauth" {
		t.Fatalf("unexpected getter values: %d %s", err.GetCode(), err.GetItem())
	}
	if err.Msg() != msg {
		t.Fatalf("unexpected msg: %#v", err.Msg())
	}

	err = WrapError(nil)
	if err != nil {
		t.Fatalf("expected nil wrap error, got %v", err)
	}

	under := errors.New("root")
	err = WrapError(under)
	if err.Error() != "root" {
		t.Fatalf("unexpected wrapped error string: %s", err.Error())
	}
	if errors.Unwrap(err) != under {
		t.Fatalf("unexpected unwrap: %v", errors.Unwrap(err))
	}

	err.SetCode(500).SetItem("server").SetMsg(msg)
	if err.Code() != 500 || err.Item() != "server" || err.Msg() != msg {
		t.Fatalf("unexpected setters: %d %s %v", err.Code(), err.Item(), err.Msg())
	}

	target := NewError(500, "server")
	if !err.Is(target) {
		t.Fatal("expected errors to match")
	}
}
