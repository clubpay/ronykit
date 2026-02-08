package utils_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
)

func TestGenericHelpers(t *testing.T) {
	var x *int
	if utils.PtrVal(x) != 0 {
		t.Fatal("expected zero value for nil pointer")
	}

	val := 10
	if utils.PtrVal(&val) != 10 {
		t.Fatal("expected pointer value")
	}

	if *utils.ValPtr(5) != 5 {
		t.Fatal("expected pointer to value")
	}

	if utils.ValPtrOrNil("") != nil {
		t.Fatal("expected nil for zero value")
	}

	if utils.ValPtrOrNil("x") == nil {
		t.Fatal("expected non-nil for non-zero value")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from Must")
		}
	}()
	_ = utils.Must(0, errors.New("boom"))
}

func TestGenericHelpersOkOrTryCast(t *testing.T) {
	if utils.OkOr(5, errors.New("nope"), 9) != 9 {
		t.Fatal("expected fallback value")
	}

	if utils.OkOr(5, nil, 9) != 5 {
		t.Fatal("expected original value")
	}

	if v := utils.TryCast[int]("nope"); v != 0 {
		t.Fatalf("expected zero value on failed cast, got %v", v)
	}

	if v := utils.TryCast[string]("ok"); v != "ok" {
		t.Fatalf("expected cast value, got %v", v)
	}

	if v := utils.Coalesce("", "a", "b"); v != "a" {
		t.Fatalf("unexpected coalesce result: %q", v)
	}

	type sample struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	m := utils.ToMap(sample{Name: "tom", Age: 30})
	if m["name"] != "tom" || m["age"] != float64(30) {
		t.Fatalf("unexpected map output: %v", m)
	}
}

func TestHashHelpers(t *testing.T) {
	input := []byte("hello")

	want256 := sha256.Sum256(input)
	got256, err := utils.Sha256(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got256, want256[:]) {
		t.Fatalf("unexpected sha256: %x", got256)
	}

	got256 = utils.MustSha256(input, nil)
	if !reflect.DeepEqual(got256, want256[:]) {
		t.Fatalf("unexpected must sha256: %x", got256)
	}

	want512 := sha512.Sum512(input)
	got512, err := utils.Sha512(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got512, want512[:]) {
		t.Fatalf("unexpected sha512: %x", got512)
	}

	got512 = utils.MustSha512(input, nil)
	if !reflect.DeepEqual(got512, want512[:]) {
		t.Fatalf("unexpected must sha512: %x", got512)
	}
}

func TestRandomHelpers(t *testing.T) {
	id := utils.RandomID(16)
	if len(id) != 16 {
		t.Fatalf("unexpected id length: %d", len(id))
	}
	for _, ch := range id {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", ch) {
			t.Fatalf("unexpected id char: %q", ch)
		}
	}

	digits := utils.RandomDigit(8)
	if len(digits) != 8 {
		t.Fatalf("unexpected digit length: %d", len(digits))
	}
	for _, ch := range digits {
		if ch < '0' || ch > '9' {
			t.Fatalf("unexpected digit char: %q", ch)
		}
	}

	if v := utils.RandomInt64(10); v < 0 || v >= 10 {
		t.Fatalf("unexpected random int64: %d", v)
	}
	if v := utils.RandomInt32(10); v < 0 || v >= 10 {
		t.Fatalf("unexpected random int32: %d", v)
	}
	if v := utils.RandomInt(10); v < 0 || v >= 10 {
		t.Fatalf("unexpected random int: %d", v)
	}
	if v := utils.RandomUint64(10); v >= 10 {
		t.Fatalf("unexpected random uint64: %d", v)
	}

	ids := utils.RandomIDs(4, 2, 3)
	if len(ids) != 3 {
		t.Fatalf("unexpected random ids length: %d", len(ids))
	}

	if v := utils.SecureRandomInt63(1); v != 0 {
		t.Fatalf("unexpected secure random int63: %d", v)
	}
	if v := utils.SecureRandomInt63(0); v < 0 {
		t.Fatalf("unexpected secure random int63: %d", v)
	}
	_ = utils.SecureRandomUint64()
}

func TestSpinLock(t *testing.T) {
	var (
		lock utils.SpinLock
		wg   sync.WaitGroup
		sum  int
	)

	inc := func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			lock.Lock()
			sum++
			lock.Unlock()
		}
	}

	wg.Add(2)
	go inc()
	go inc()
	wg.Wait()

	if sum != 2000 {
		t.Fatalf("unexpected sum: %d", sum)
	}
}

func TestTimeHelpers(t *testing.T) {
	now := time.Now().Unix()
	got := utils.TimeUnix()
	if got < now-2 || got > now+2 {
		t.Fatalf("unexpected TimeUnix value: %d", got)
	}

	if v := utils.TimeUnixAdd(100, 5*time.Second); v != 105 {
		t.Fatalf("unexpected TimeUnixAdd: %d", v)
	}
	if v := utils.TimeUnixSubtract(100, 3*time.Second); v != 97 {
		t.Fatalf("unexpected TimeUnixSubtract: %d", v)
	}

	if utils.NanoTime() <= 0 {
		t.Fatal("expected nanotime to be positive")
	}
	if utils.CPUTicks() <= 0 {
		t.Fatal("expected cputicks to be positive")
	}
}

func TestTransformHelpers(t *testing.T) {
	if utils.ToCamel("hello_world") != "HelloWorld" {
		t.Fatalf("unexpected ToCamel")
	}
	if utils.ToLowerCamel("ID") != "id" {
		t.Fatalf("unexpected ToLowerCamel for acronym")
	}
	if utils.ToSnake("HTTPServer") != "http_server" {
		t.Fatalf("unexpected ToSnake")
	}
	if utils.ToScreamingSnake("HTTPServer") != "HTTP_SERVER" {
		t.Fatalf("unexpected ToScreamingSnake")
	}
	if utils.ToKebab("HelloWorld") != "hello-world" {
		t.Fatalf("unexpected ToKebab")
	}
	if utils.ToScreamingKebab("HelloWorld") != "HELLO-WORLD" {
		t.Fatalf("unexpected ToScreamingKebab")
	}
	if utils.ToDelimited("HelloWorld", '.') != "hello.world" {
		t.Fatalf("unexpected ToDelimited")
	}
	if utils.ToSnakeWithIgnore("a b", ' ') != "a b" {
		t.Fatalf("unexpected ToSnakeWithIgnore")
	}
}

func TestVisitorHelpers(t *testing.T) {
	state := utils.VisitAll(1, func(v *int) { *v += 2 }, func(v *int) { *v *= 3 })
	if state != 9 {
		t.Fatalf("unexpected VisitAll result: %d", state)
	}

	condState := utils.VisitCond(1, func(v *int) bool { return *v < 2 }, func(v *int) { *v += 1 }, func(v *int) { *v += 10 })
	if condState != 2 {
		t.Fatalf("unexpected VisitCond result: %d", condState)
	}

	stopState, err := utils.VisitStopOnErr(1,
		func(v *int) error { *v += 1; return nil },
		func(v *int) error { return errors.New("stop") },
		func(v *int) error { *v += 10; return nil },
	)
	if err == nil || stopState != 2 {
		t.Fatalf("unexpected VisitStopOnErr result: %v, %d", err, stopState)
	}
}

func TestConvertHelpers(t *testing.T) {
	if utils.StrToFloat64("1.5") != 1.5 {
		t.Fatal("unexpected StrToFloat64")
	}
	if utils.StrToFloat32("2.5") != float32(2.5) {
		t.Fatal("unexpected StrToFloat32")
	}
	if utils.StrToInt64("10") != 10 {
		t.Fatal("unexpected StrToInt64")
	}
	if utils.StrToInt32("11") != 11 {
		t.Fatal("unexpected StrToInt32")
	}
	if utils.StrToUInt64("12") != 12 {
		t.Fatal("unexpected StrToUInt64")
	}
	if utils.StrToUInt32("13") != 13 {
		t.Fatal("unexpected StrToUInt32")
	}
	if utils.StrToInt("14") != 14 {
		t.Fatal("unexpected StrToInt")
	}
	if utils.StrToUInt("15") != 15 {
		t.Fatal("unexpected StrToUInt")
	}
	if utils.Int64ToStr(16) != "16" {
		t.Fatal("unexpected Int64ToStr")
	}
	if utils.Int32ToStr(17) != "17" {
		t.Fatal("unexpected Int32ToStr")
	}
	if utils.UInt64ToStr(18) != "18" {
		t.Fatal("unexpected UInt64ToStr")
	}
	if utils.UInt32ToStr(19) != "19" {
		t.Fatal("unexpected UInt32ToStr")
	}
	if utils.Float64ToStr(1.25) != "1.25" {
		t.Fatal("unexpected Float64ToStr")
	}
	if utils.F64ToStr(1.5) != "1.5" {
		t.Fatal("unexpected F64ToStr")
	}
	if utils.Float32ToStr(2.25) != "2.25" {
		t.Fatal("unexpected Float32ToStr")
	}
	if utils.F32ToStr(2.5) != "2.5" {
		t.Fatal("unexpected F32ToStr")
	}
	if utils.IntToStr(20) != "20" {
		t.Fatal("unexpected IntToStr")
	}
	if utils.UIntToStr(21) != "21" {
		t.Fatal("unexpected UIntToStr")
	}

	n := utils.ParseNumeric("13.14").WithoutPrecision()
	if n.String() != "13.14" {
		t.Fatalf("unexpected without precision: %s", n.String())
	}
}

func TestAppendStrHelpers(t *testing.T) {
	var sb strings.Builder
	utils.AppendStrInt(&sb, 1)
	if sb.Len() != 8 {
		t.Fatalf("unexpected AppendStrInt length: %d", sb.Len())
	}

	sb.Reset()
	utils.AppendStrUInt(&sb, 1)
	if sb.Len() != 8 {
		t.Fatalf("unexpected AppendStrUInt length: %d", sb.Len())
	}

	sb.Reset()
	utils.AppendStrInt64(&sb, 1)
	if sb.Len() != 8 {
		t.Fatalf("unexpected AppendStrInt64 length: %d", sb.Len())
	}

	sb.Reset()
	utils.AppendStrUInt64(&sb, 1)
	if sb.Len() != 8 {
		t.Fatalf("unexpected AppendStrUInt64 length: %d", sb.Len())
	}

	sb.Reset()
	utils.AppendStrInt32(&sb, 1)
	if sb.Len() != 4 {
		t.Fatalf("unexpected AppendStrInt32 length: %d", sb.Len())
	}

	sb.Reset()
	utils.AppendStrUInt32(&sb, 1)
	if sb.Len() != 4 {
		t.Fatalf("unexpected AppendStrUInt32 length: %d", sb.Len())
	}
}

func TestUnsafeConversions(t *testing.T) {
	b := []byte("hello")
	if utils.ByteToStr(b) != "hello" {
		t.Fatalf("unexpected ByteToStr")
	}
	if string(utils.StrToByte("world")) != "world" {
		t.Fatalf("unexpected StrToByte")
	}
}
