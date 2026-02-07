package logkit

import (
	"reflect"
	"testing"
)

type sensitiveStruct struct {
	Password string  `sensitive:"true"`
	Email    string  `sensitive:"email"`
	Phone    *string `sensitive:"phone"`
	Public   string
}

func TestMaskStruct(t *testing.T) {
	phone := "1234567890"
	in := sensitiveStruct{
		Password: "secret",
		Email:    "user@example.com",
		Phone:    &phone,
		Public:   "ok",
	}

	out := maskStruct(reflect.ValueOf(in)).(sensitiveStruct)
	if out.Password != "" {
		t.Fatalf("expected password to be masked")
	}
	if out.Email == "user@example.com" {
		t.Fatalf("expected email to be masked")
	}
	if out.Phone == nil || *out.Phone == phone {
		t.Fatalf("expected phone to be masked")
	}
	if out.Public != "ok" {
		t.Fatalf("unexpected public field: %q", out.Public)
	}
}
