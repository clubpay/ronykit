package errs_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/stub"
)

func TestError(t *testing.T) {
	err1 := errs.B().Code(errs.InvalidArgument).Msg("SOME_ERROR").Err()
	err1JSON, err := json.Marshal(err1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("err1:", err1)
	t.Log("err1JSON:", string(err1JSON))
	err2 := stub.NewError(err1.(interface{ GetCode() int }).GetCode(), string(err1JSON))
	t.Log("err2:", err2)

	err3 := errs.Convert(err2)
	t.Log("err3", err3)
	err3JSON, err := json.Marshal(err1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("err3JSON:", string(err3JSON))

	err4 := stub.NewError(errs.InvalidArgument.HTTPStatus(), "SOME_ERROR")
	cErr4 := errs.Convert(err4)
	var cErr40 *errs.Error
	if !errors.As(cErr4, &cErr40) {
		t.Fatal("not a error")
	}
	t.Log("cErr4", cErr40.GetCode(), cErr40.GetItem())
	err4JSON, err := json.Marshal(cErr4)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("err4JSON:", string(err4JSON))

	err5JSON := `{"code":"not_found","item":"ACCOUNT_NOT_FOUND","details":null}`
	err5 := stub.NewError(http.StatusConflict, err5JSON)
	t.Log(err5.Code(), err5.Item())
	cErr5 := errs.Convert(err5)
	var cErr50 *errs.Error
	if !errors.As(cErr5, &cErr50) {
		t.Fatal("not a error")
	}
	t.Log("cErr5", cErr50.GetCode(), cErr50.GetItem())
	err5JSON2, err := json.Marshal(cErr50)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("err5JSON2:", string(err5JSON2))
}

func TestCodeNamePopulated(t *testing.T) {
	t.Parallel()

	// The outward REST wire marshals the Error struct directly, so a missing
	// CodeName used to serialize as "" — fill it at every construction site.
	built := errs.B().Code(errs.NotFound).Msg("missing").Err()
	e := &errs.Error{}
	if !errors.As(built, &e) || e.CodeName != "not_found" {
		t.Fatalf("builder CodeName = %q, want not_found", e.CodeName)
	}

	wrapped := errs.Wrap(built, "outer")
	e = &errs.Error{}
	if !errors.As(wrapped, &e) || e.CodeName != "not_found" {
		t.Fatalf("wrap CodeName = %q, want not_found", e.CodeName)
	}

	wrappedCode := errs.WrapCode(built, errs.AlreadyExists, "outer")
	e = &errs.Error{}
	if !errors.As(wrappedCode, &e) || e.CodeName != "already_exists" {
		t.Fatalf("wrapCode CodeName = %q, want already_exists", e.CodeName)
	}

	conv := errs.Convert(errors.New("plain"))
	e = &errs.Error{}
	if !errors.As(conv, &e) || e.CodeName != "unknown" {
		t.Fatalf("convert CodeName = %q, want unknown", e.CodeName)
	}
}
