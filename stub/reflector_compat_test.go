package stub_test

import (
	"testing"

	kitreflector "github.com/clubpay/ronykit/kit/utils/reflector"
	rkitreflector "github.com/clubpay/ronykit/x/rkit/reflector"
)

type compatMsg struct {
	embeddedCompat

	Name   string `json:"name"`
	Count  int64  `json:"count"`
	hidden string
}

type embeddedCompat struct {
	Code int `json:"code"`
}

func TestReflectorMatchesKitUtils(t *testing.T) {
	msg := &compatMsg{
		embeddedCompat: embeddedCompat{Code: 7},
		Name:           "alice",
		Count:          42,
		hidden:         "nope",
	}

	kitreflector.Register(msg, "json")
	rkitreflector.Register(msg, "json")

	kitR := kitreflector.New()
	rkitR := rkitreflector.New()

	kitRef := kitR.Load(msg, "json")
	rkitRef := rkitR.Load(msg, "json")

	kitObj := kitRef.Obj()
	rkitObj := rkitRef.Obj()

	if kitObj.GetStringDefault(msg, "Name", "") != rkitObj.GetStringDefault(msg, "Name", "") {
		t.Fatal("GetStringDefault Name mismatch")
	}
	if kitObj.GetInt64Default(msg, "Count", 0) != rkitObj.GetInt64Default(msg, "Count", 0) {
		t.Fatal("GetInt64Default Count mismatch")
	}
	if kitObj.GetIntDefault(msg, "Code", 0) != rkitObj.GetIntDefault(msg, "Code", 0) {
		t.Fatal("GetIntDefault Code mismatch")
	}
	if kitObj.Get(msg, "Name") != rkitObj.Get(msg, "Name") {
		t.Fatalf("Get Name mismatch: %v vs %v", kitObj.Get(msg, "Name"), rkitObj.Get(msg, "Name"))
	}

	kitByTag, kitOK := kitRef.ByTag("json")
	rkitByTag, rkitOK := rkitRef.ByTag("json")
	if kitOK != rkitOK {
		t.Fatalf("ByTag ok mismatch: %v vs %v", kitOK, rkitOK)
	}
	if kitByTag.GetStringDefault(msg, "name", "") != rkitByTag.GetStringDefault(msg, "name", "") {
		t.Fatal("ByTag name mismatch")
	}
	if kitByTag.GetInt64Default(msg, "count", 0) != rkitByTag.GetInt64Default(msg, "count", 0) {
		t.Fatal("ByTag count mismatch")
	}
	if kitByTag.GetIntDefault(msg, "code", 0) != rkitByTag.GetIntDefault(msg, "code", 0) {
		t.Fatal("ByTag code mismatch")
	}

	kitName, kitErr := kitR.GetString(msg, "Name")
	rkitName, rkitErr := rkitR.GetString(msg, "Name")
	if kitErr != nil || rkitErr != nil || kitName != rkitName {
		t.Fatalf("GetString mismatch: %q %v vs %q %v", kitName, kitErr, rkitName, rkitErr)
	}

	kitCount, kitErr := kitR.GetInt(msg, "Count")
	rkitCount, rkitErr := rkitR.GetInt(msg, "Count")
	if kitErr != nil || rkitErr != nil || kitCount != rkitCount {
		t.Fatalf("GetInt mismatch: %d %v vs %d %v", kitCount, kitErr, rkitCount, rkitErr)
	}

	if _, err := rkitR.GetString(msg, "hidden"); err != rkitreflector.ErrNotExported {
		t.Fatalf("expected ErrNotExported, got %v", err)
	}
	if _, err := kitR.GetString(msg, "hidden"); err != kitreflector.ErrNotExported {
		t.Fatalf("kit expected ErrNotExported, got %v", err)
	}
}
