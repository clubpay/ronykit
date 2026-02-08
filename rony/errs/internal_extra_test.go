package errs

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/rony/errs/errmarshalling"
	jsoniter "github.com/json-iterator/go"
)

func TestInternalStreamMarshalling(t *testing.T) {
	err := &Error{
		Code:       NotFound,
		Item:       "missing",
		Meta:       Metadata{"k": "v"},
		underlying: errors.New("root"),
	}

	var buf bytes.Buffer
	stream := jsoniter.NewStream(jsoniter.ConfigCompatibleWithStandardLibrary, &buf, 128)
	stream.WriteObjectStart()
	writeErrorFieldsToInternalStream(err, stream)
	stream.WriteObjectEnd()
	if err := stream.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"code"`) || !strings.Contains(buf.String(), `"item"`) {
		t.Fatalf("unexpected stream output: %s", buf.String())
	}
}

func TestJSONIterEncoder(t *testing.T) {
	e := Error{Code: NotFound, Item: "missing"}
	data, err := jsoniter.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"code"`) || !strings.Contains(string(data), `"message"`) {
		t.Fatalf("unexpected json output: %s", string(data))
	}
}

func TestInternalIteratorUnmarshal(t *testing.T) {
	wrapped := errmarshalling.Marshal(errors.New("root"))
	data := fmt.Sprintf(`{"code":5,"codeName":"not_found","item":"missing","meta":{"k":"v"},"wraps":%s}`, string(wrapped))

	itr := jsoniter.ConfigCompatibleWithStandardLibrary.BorrowIterator([]byte(data))
	defer jsoniter.ConfigCompatibleWithStandardLibrary.ReturnIterator(itr)

	out := &Error{}
	unmarshalFromInternalIterator(out, itr)
	if itr.Error != nil {
		t.Fatal(itr.Error)
	}
	if out.Code != NotFound || out.Item != "missing" {
		t.Fatalf("unexpected decoded error: %+v", out)
	}
	if out.Meta["k"] != "v" {
		t.Fatalf("unexpected decoded meta: %v", out.Meta)
	}
	if out.underlying == nil || out.underlying.Error() != "root" {
		t.Fatalf("unexpected decoded underlying: %v", out.underlying)
	}
}
