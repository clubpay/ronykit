package errs_test

import (
	"encoding/json"
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
}
