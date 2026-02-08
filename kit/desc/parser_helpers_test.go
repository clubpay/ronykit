package desc_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type SampleError struct {
	Code  int    `json:"code"`
	Item  string `json:"item"`
	Extra string `json:"extra"`
}

func (s SampleError) Error() string { return s.Item }
func (s SampleError) GetCode() int  { return s.Code }
func (s SampleError) GetItem() string {
	return s.Item
}

type MixedMessage struct {
	Public  string `json:"public"`
	private string `json:"private"`
}

func TestParsedContractHelpers(t *testing.T) {
	pc := desc.ParsedContract{
		GroupName: "group",
		Index:     2,
		Type:      desc.REST,
		Path:      "/items/:id",
	}
	if pc.SuggestName() != "Items" {
		t.Fatalf("unexpected suggest name: %s", pc.SuggestName())
	}

	pc = desc.ParsedContract{
		Name:      "explicit",
		GroupName: "group",
		Index:     2,
		Type:      desc.REST,
		Path:      "/x",
	}
	if pc.SuggestName() != "explicit" {
		t.Fatalf("unexpected suggest name: %s", pc.SuggestName())
	}

	pc = desc.ParsedContract{
		GroupName: "group",
		Index:     3,
		Type:      desc.RPC,
		Predicate: "get_item",
	}
	if pc.SuggestName() != "GetItem" {
		t.Fatalf("unexpected suggest name: %s", pc.SuggestName())
	}

	pc = desc.ParsedContract{
		GroupName: "group",
		Index:     1,
		Type:      "",
		Path:      "/:id",
	}
	if pc.SuggestName() != "group1" {
		t.Fatalf("unexpected suggest name: %s", pc.SuggestName())
	}

	pc.Responses = []desc.ParsedResponse{
		{ErrCode: 400},
		{ErrCode: 0, ErrItem: "ok"},
	}
	okResp := pc.OKResponse()
	if okResp.IsError() || okResp.ErrItem != "ok" {
		t.Fatalf("unexpected ok response: %+v", okResp)
	}

	pc.PathParams = []string{"id"}
	if !pc.IsPathParam("id") {
		t.Fatal("expected path param to match")
	}
}

func TestParsedMessageHelpers(t *testing.T) {
	svc := desc.NewService("svc").AddContract(
		desc.NewContract().
			SetName("c1").
			AddRoute(desc.Route("", newREST(kit.JSON, "/items/{id}", "GET"))).
			In(&MixedMessage{}).
			Out(&SampleError{}),
	)

	pd := desc.ParseService(svc)
	errMsg := pd.MessageByName("SampleError")
	if errMsg == nil {
		t.Fatal("expected SampleError message")
	}
	if !errMsg.ImplementError {
		t.Fatal("expected error message to implement error")
	}
	if errMsg.CodeField() != "Code" {
		t.Fatalf("unexpected code field: %s", errMsg.CodeField())
	}
	if errMsg.ItemField() != "Item" {
		t.Fatalf("unexpected item field: %s", errMsg.ItemField())
	}
	if !strings.Contains(errMsg.JSON(), "\"code\"") {
		t.Fatalf("unexpected json output: %s", errMsg.JSON())
	}
	if !strings.Contains(errMsg.String(), "code:integer") {
		t.Fatalf("unexpected string output: %s", errMsg.String())
	}

	mixedMsg := pd.MessageByName("MixedMessage")
	if mixedMsg == nil {
		t.Fatal("expected MixedMessage")
	}
	if mixedMsg.FieldByName("public") == nil {
		t.Fatal("expected field by json name")
	}
	if mixedMsg.FieldByGoName("Public") == nil {
		t.Fatal("expected field by go name")
	}
	if mixedMsg.TotalExportedFields() != 1 || len(mixedMsg.ExportedFields()) != 1 {
		t.Fatalf("unexpected exported fields: %d", mixedMsg.TotalExportedFields())
	}
}

func TestParsedMessageSpecial(t *testing.T) {
	pm := desc.ParsedMessage{Kind: desc.KitRawMessage}
	if !pm.IsSpecial() || pm.GoName() != "kit.RawMessage" {
		t.Fatalf("unexpected special raw message: %+v", pm)
	}

	pm = desc.ParsedMessage{Kind: desc.KitMultipartFormMessage}
	if !pm.IsSpecial() || pm.GoName() != "kit.MultipartFormMessage" {
		t.Fatalf("unexpected special multipart message: %+v", pm)
	}
}

func TestParsedStructTagHelpers(t *testing.T) {
	pst := desc.ParsedStructTag{
		Raw: reflect.StructTag(`json:"a" xml:"b"`),
	}
	tags := pst.Tags("json", "xml")
	if tags["json"] != "a" || tags["xml"] != "b" {
		t.Fatalf("unexpected tags: %v", tags)
	}
	if pst.Get("json") != "a" {
		t.Fatalf("unexpected tag get: %s", pst.Get("json"))
	}
}

func TestParsedElementString(t *testing.T) {
	base := desc.ParsedElement{Kind: desc.String}
	arr := desc.ParsedElement{Kind: desc.Array, Element: &base}
	if arr.String() != "array[string]" {
		t.Fatalf("unexpected array string: %s", arr.String())
	}

	m := desc.ParsedElement{Kind: desc.Map, Element: &base}
	if m.String() != "map[string]" {
		t.Fatalf("unexpected map string: %s", m.String())
	}

	pm := desc.ParsedMessage{
		Name: "Obj",
		Fields: []desc.ParsedField{
			{Name: "a", Element: &base},
		},
	}
	obj := desc.ParsedElement{Kind: desc.Object, Message: &pm}
	if !strings.HasPrefix(obj.String(), "Obj[") {
		t.Fatalf("unexpected object string: %s", obj.String())
	}
}
