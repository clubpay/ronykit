package fasthttp

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

type ptrMessage struct {
	Bool    bool     `json:"bool"`
	BoolPtr *bool    `json:"boolPtr"`
	Str     string   `json:"str"`
	StrPtr  *string  `json:"strPtr"`
	I64     int64    `json:"i64"`
	I64Ptr  *int64   `json:"i64Ptr"`
	U32     uint32   `json:"u32"`
	U32Ptr  *uint32  `json:"u32Ptr"`
	F32     float32  `json:"f32"`
	F32Ptr  *float32 `json:"f32Ptr"`
	Ints    []int64  `json:"ints"`
	Bytes   []byte   `json:"bytes"`
	Strings []string `json:"strings"`
}

func TestReflectDecoderPointersAndSlices(t *testing.T) {
	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&ptrMessage{}))
	ctx := newRequestCtx(MethodPost, "/decode")

	ctx.SetUserValue("str", "user")
	ctx.Request.URI().SetQueryString("bool=true&boolPtr=true&strPtr=query&i64=42&i64Ptr=84&f32=1.5&f32Ptr=2.5&ints=1&ints=2&bytes=abc&strings=a&strings=b")
	ctx.Request.PostArgs().Set("u32", "7")
	ctx.Request.PostArgs().Set("u32Ptr", "9")

	msg, err := dec(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*ptrMessage) //nolint:forcetypeassert
	if !m.Bool || m.BoolPtr == nil || !*m.BoolPtr {
		t.Fatalf("bool fields not decoded")
	}
	if m.Str != "user" || m.StrPtr == nil || *m.StrPtr != "query" {
		t.Fatalf("string fields not decoded: %q %v", m.Str, m.StrPtr)
	}
	if m.I64 != 42 || m.I64Ptr == nil || *m.I64Ptr != 84 {
		t.Fatalf("int64 fields not decoded")
	}
	if m.U32 != 7 || m.U32Ptr == nil || *m.U32Ptr != 9 {
		t.Fatalf("uint32 fields not decoded")
	}
	if m.F32 != float32(1.5) || m.F32Ptr == nil || *m.F32Ptr != float32(2.5) {
		t.Fatalf("float32 fields not decoded")
	}
	if len(m.Ints) != 2 || m.Ints[0] != int64(1) || m.Ints[1] != int64(2) {
		t.Fatalf("slice fields not decoded: %v", m.Ints)
	}
	if string(m.Bytes) != "abc" {
		t.Fatalf("bytes field not decoded: %q", string(m.Bytes))
	}
	if len(m.Strings) != 2 || m.Strings[0] != "a" || m.Strings[1] != "b" {
		t.Fatalf("strings field not decoded: %v", m.Strings)
	}
}

func TestReflectDecoderRawMessage(t *testing.T) {
	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(kit.RawMessage{}))
	ctx := newRequestCtx(MethodPost, "/raw")
	msg, err := dec(ctx, []byte("raw"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(msg.(kit.RawMessage)) != "raw" { //nolint:forcetypeassert
		t.Fatalf("unexpected raw message: %v", msg)
	}
}

func TestReflectDecoderMultipartForm(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("a", "b")
	_ = writer.Close()

	ctx := newRequestCtx(MethodPost, "/multipart")
	ctx.Request.SetBodyRaw(body.Bytes())
	ctx.Request.Header.SetContentType(writer.FormDataContentType())

	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&kit.MultipartFormMessage{}))
	msg, err := dec(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(kit.MultipartFormMessage) //nolint:forcetypeassert
	if m.GetForm() == nil || m.GetForm().Value["a"][0] != "b" {
		t.Fatalf("unexpected multipart form content")
	}
}

func TestReflectDecoderInvalidInput(t *testing.T) {
	assertPanic := func(fn func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()
		fn()
	}

	assertPanic(func() {
		reflectDecoder(kit.JSON, func() kit.Message { return ptrMessage{} })
	})

	assertPanic(func() {
		reflectDecoder(kit.JSON, func() kit.Message { return &[]int{} })
	})
}

func TestGetParamsTrimAndUserValues(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue("a", "b")
	ctx.Request.URI().SetQueryString("tags[]=x&tags[]=y")
	ctx.Request.PostArgs().Set("post", "ok")

	params := GetParams(ctx)
	var seenTags int
	var sawUser bool
	var sawPost bool
	for _, p := range params {
		switch p.Key {
		case "tags":
			seenTags++
		case "a":
			sawUser = true
		case "post":
			sawPost = true
		}
	}
	if seenTags != 2 || !sawUser || !sawPost {
		t.Fatalf("unexpected params: %+v", params)
	}
}
