package desc_test

import (
	"reflect"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyRESTSelector struct {
	enc    kit.Encoding
	path   string
	method string
}

func newREST(enc kit.Encoding, path, method string) dummyRESTSelector {
	return dummyRESTSelector{
		enc:    enc,
		path:   path,
		method: method,
	}
}

func (d dummyRESTSelector) Query(_ string) any {
	return nil
}

func (d dummyRESTSelector) GetEncoding() kit.Encoding {
	return d.enc
}

func (d dummyRESTSelector) GetMethod() string {
	return d.method
}

func (d dummyRESTSelector) GetPath() string {
	return d.path
}

func (d dummyRESTSelector) String() string {
	return d.method + " " + d.path
}

type FlatMessage struct {
	A string                       `json:"a"`
	B int64                        `json:"b"`
	C map[string]string            `json:"c"`
	D map[string]int64             `json:"d"`
	E map[int64]string             `json:"e"`
	F map[string]any               `json:"f"`
	G [][]string                   `json:"g"`
	M map[string]map[string]string `json:"m"`
	T time.Time                    `json:"t"`
}

type NestedMessage struct {
	A    string                  `json:"a"`
	B    FlatMessage             `json:"b"`
	BA   []FlatMessage           `json:"ba,omitempty"`
	BM   map[string]FlatMessage  `json:"bm"`
	C    *FlatMessage            `json:"c"`
	PA   []*FlatMessage          `json:"pa"`
	PMap map[string]*FlatMessage `json:"pmap"`
}

type Err struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var _ kit.ErrorMessage = (*Err)(nil)

func (e Err) Error() string {
	return e.Msg
}

func (e Err) GetCode() int {
	return e.Code
}

func (e Err) GetItem() string {
	return e.Msg
}

type TwoFields struct {
	A string `json:"a"`
	B string `json:"b"`
}

type PtrErr struct {
	Code int    `json:"code"`
	Item string `json:"item"`
}

var _ kit.ErrorMessage = (*PtrErr)(nil)

func (e *PtrErr) Error() string {
	return e.Item
}

func (e *PtrErr) GetCode() int {
	return e.Code
}

func (e *PtrErr) GetItem() string {
	return e.Item
}

func TestDescParser(t *testing.T) {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("c1").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/path2", "POST"))).
				SetDefaultError(&Err{}).
				In(&NestedMessage{}).
				Out(&FlatMessage{}),
		)

	pd := desc.ParseService(d)
	contract0 := pd.Contracts[0]
	contract1 := pd.Contracts[1]
	assert.Len(t, pd.Contracts, 2)
	assert.Len(t, pd.Messages(), 3)
	assert.NotNil(t, pd.MessageByName("Err"))
	assert.True(t, pd.MessageByName("Err").ImplementError)

	assert.Equal(t, "s1", contract0.Name)
	assert.Equal(t, desc.REST, contract0.Type)
	assert.Equal(t, kit.JSON.Tag(), contract0.Encoding)
	assert.Equal(t, "/path1", contract0.Path)
	assert.Equal(t, "GET", contract0.Method)
	assert.Equal(t, "c1", contract0.GroupName)
	assert.Equal(t, "NestedMessage", contract0.Request.Message.Name)
	assert.Len(t, contract0.Request.Message.Fields, 7)
	assert.Equal(t, "FlatMessage", contract0.Responses[0].Message.Name)
	assert.Len(t, contract0.Responses[0].Message.Fields, 9)

	assert.Equal(t, "s2", contract1.Name)
	assert.Equal(t, desc.REST, contract1.Type)
	assert.Equal(t, kit.JSON.Tag(), contract1.Encoding)
	assert.Equal(t, "/path2", contract1.Path)
	assert.Equal(t, "POST", contract1.Method)
	assert.Equal(t, "c1", contract1.GroupName)

	assert.Equal(t, "NestedMessage", contract0.Request.Message.Name)
	assert.Len(t, contract0.Request.Message.Fields, 7)
	assert.Equal(t, "a", contract0.Request.Message.Fields[0].Name)
	assert.Equal(t, desc.String, contract0.Request.Message.Fields[0].Element.Kind)
	assert.Equal(t, "b", contract0.Request.Message.Fields[1].Name)
	assert.Equal(t, desc.Object, contract0.Request.Message.Fields[1].Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.Fields[1].Element.Message.Name)
	assert.False(t, contract0.Request.Message.Fields[1].Optional)
	assert.Equal(t, "ba", contract0.Request.Message.Fields[2].Name)
	assert.Equal(t, desc.Array, contract0.Request.Message.Fields[2].Element.Kind)
	assert.Equal(t, desc.Object, contract0.Request.Message.Fields[2].Element.Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.Fields[2].Element.Element.Message.Name)
	assert.True(t, contract0.Request.Message.Fields[2].Tag.OmitEmpty)
	assert.True(t, contract0.Request.Message.Fields[2].Optional)
	assert.Equal(t, "bm", contract0.Request.Message.Fields[3].Name)
	assert.Equal(t, desc.Map, contract0.Request.Message.Fields[3].Element.Kind)
	assert.Equal(t, desc.Object, contract0.Request.Message.Fields[3].Element.Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.Fields[3].Element.Element.Message.Name)
	assert.True(t, contract0.Request.Message.Fields[3].Optional)
	assert.False(t, contract0.Request.Message.Fields[3].Tag.OmitEmpty)
	assert.Equal(t, "c", contract0.Request.Message.Fields[4].Name)
	assert.Equal(t, desc.Object, contract0.Request.Message.Fields[4].Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.Fields[4].Element.Message.Name)
	assert.True(t, contract0.Request.Message.Fields[4].Optional)
	assert.Equal(t, desc.Array, contract0.Request.Message.FieldByName("ba").Element.Kind)
	assert.Equal(t, desc.Object, contract0.Request.Message.FieldByGoName("BA").Element.Element.Kind)
	assert.Equal(t, desc.Array, contract0.Request.Message.FieldByGoName("PA").Element.Kind)
	assert.Equal(t, desc.Object, contract0.Request.Message.FieldByGoName("PA").Element.Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.FieldByGoName("PA").Element.Element.Message.Name)
	assert.Equal(t, desc.Map, contract0.Request.Message.FieldByGoName("PMap").Element.Kind)
	assert.Equal(t, desc.Object, contract0.Request.Message.FieldByGoName("PMap").Element.Element.Kind)
	assert.Equal(t, "FlatMessage", contract0.Request.Message.FieldByGoName("PMap").Element.Element.Message.Name)

	b := contract0.Request.Message.Fields[1]
	assert.Equal(t, "b", b.Name)
	assert.Equal(t, desc.Object, b.Element.Kind)
	assert.Equal(t, "FlatMessage", b.Element.Message.Name)
	assert.Len(t, b.Element.Message.Fields, 9)
	assert.Equal(t, "a", b.Element.Message.Fields[0].Name)
	assert.Equal(t, desc.String, b.Element.Message.Fields[0].Element.Kind)

	ba := contract0.Request.Message.Fields[2]
	assert.Equal(t, "ba", ba.Name)
	assert.Equal(t, desc.Array, ba.Element.Kind)
	assert.Nil(t, ba.Element.Message)
	assert.Equal(t, desc.Object, ba.Element.Element.Kind)
	assert.Equal(t, "FlatMessage", ba.Element.Element.Message.Name)
	assert.Len(t, ba.Element.Element.Message.Fields, 9)
	assert.Equal(t, "a", ba.Element.Element.Message.Fields[0].Name)
	assert.Equal(t, desc.String, ba.Element.Element.Message.Fields[0].Element.Kind)

	cField := contract0.Responses[0].Message.Fields[2]
	assert.Equal(t, "c", cField.Name)
	assert.Equal(t, desc.Map, cField.Element.Kind)
	assert.Equal(t, desc.String, cField.Element.Element.Kind)
	assert.Nil(t, cField.Element.Message)
	assert.True(t, cField.Optional)
	assert.Nil(t, cField.Element.Message)
	assert.Equal(t, "map[string]string", cField.Element.Type)

	dField := contract0.Responses[0].Message.Fields[3]
	assert.Equal(t, "d", dField.Name)
	assert.Equal(t, desc.Map, dField.Element.Kind)
	assert.Equal(t, desc.Integer, dField.Element.Element.Kind)
	assert.Nil(t, dField.Element.Message)
	assert.True(t, dField.Optional)
	assert.Nil(t, dField.Element.Message)
	assert.Equal(t, "map[string]int64", dField.Element.Type)

	fField := contract0.Responses[0].Message.Fields[5]
	assert.Equal(t, "f", fField.Name)
	assert.Equal(t, desc.Map, fField.Element.Kind)
	assert.Nil(t, fField.Element.Message)
	assert.True(t, fField.Optional)
	assert.Nil(t, fField.Element.Message)
	assert.Equal(t, "map[string]any", fField.Element.Type)

	gField := contract0.Responses[0].Message.Fields[6]
	assert.Equal(t, "g", gField.Name)
	assert.Equal(t, desc.Array, gField.Element.Kind)
	assert.Equal(t, desc.Array, gField.Element.Element.Kind)
	assert.Equal(t, desc.String, gField.Element.Element.Element.Kind)
	assert.Nil(t, gField.Element.Element.Element.Message)
	assert.Nil(t, gField.Element.Element.Message)

	mField := contract0.Responses[0].Message.Fields[7]
	assert.Equal(t, "m", mField.Name)
	assert.Equal(t, desc.Map, mField.Element.Kind)
	assert.Equal(t, desc.Map, mField.Element.Element.Kind)
	assert.Nil(t, mField.Element.Element.Message)
	assert.Equal(t, desc.String, mField.Element.Element.Element.Kind)
	assert.Nil(t, mField.Element.Element.Element.Message)

}

func TestDescParserEdgeCases(t *testing.T) {
	t.Run("should return the actual field entry", func(t *testing.T) {
		d := desc.NewService("sample").
			AddContract(
				desc.NewContract().
					SetName("c1").
					AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
					In(&TwoFields{}).
					Out(&FlatMessage{}),
			)

		pd := desc.ParseService(d)
		msg := pd.Contracts[0].Request.Message

		fieldByName := msg.FieldByName("a")
		assert.NotNil(t, fieldByName)
		assert.Equal(t, &msg.Fields[0], fieldByName)
		assert.Equal(t, "A", fieldByName.GoName)

		fieldByGoName := msg.FieldByGoName("A")
		assert.NotNil(t, fieldByGoName)
		assert.Equal(t, &msg.Fields[0], fieldByGoName)
		assert.Equal(t, "a", fieldByGoName.Name)
	})

	t.Run("should mark pointer-receiver errors as implementing ErrorMessage", func(t *testing.T) {
		d := desc.NewService("sample").
			AddContract(
				desc.NewContract().
					SetName("c1").
					AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
					SetDefaultError(&PtrErr{}).
					In(&FlatMessage{}).
					Out(&FlatMessage{}),
			)

		pd := desc.ParseService(d)
		errMsg := pd.MessageByName("PtrErr")
		assert.NotNil(t, errMsg)
		assert.True(t, errMsg.ImplementError)
	})
}

func TestParseMessageJSON(t *testing.T) {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetInputHeader(
					desc.RequiredHeader("hdr1"),
					desc.OptionalHeader("optionalHdr1"),
				).
				SetName("c1").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/path2", "POST"))).
				In(&NestedMessage{}).
				Out(&FlatMessage{}),
		)

	ps := desc.ParseService(d)
	assert.Len(t, ps.Messages(), 2)
	assert.Len(t, ps.Contracts, 2)
	assert.Equal(t, desc.REST, ps.Contracts[0].Type)
	assert.Len(t, ps.Contracts[0].Request.Headers, 2)
	assert.True(t, ps.Contracts[0].Request.Headers[0].Required)
	assert.Equal(t, "hdr1", ps.Contracts[0].Request.Headers[0].Name)
	assert.False(t, ps.Contracts[0].Request.Headers[1].Required)
	assert.Equal(t, "optionalHdr1", ps.Contracts[0].Request.Headers[1].Name)
}

func TestRawMessageAndMultipartForm(t *testing.T) {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("rawRequest").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/raw1", "POST"))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}),
		).
		AddContract(
			desc.NewContract().
				SetName("multipartFormRequest").
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/multipart/form", "POST"))).
				In(kit.MultipartFormMessage{}).
				Out(kit.RawMessage{}),
		)

	pd := desc.ParseService(d)
	contract0 := pd.Contracts[0]
	contract1 := pd.Contracts[1]
	assert.Len(t, pd.Contracts, 2)

	assert.Equal(t, "s1", contract0.Name)
	assert.Equal(t, desc.REST, contract0.Type)
	assert.Equal(t, kit.JSON.Tag(), contract0.Encoding)
	assert.Equal(t, "/raw1", contract0.Path)
	assert.Equal(t, "POST", contract0.Method)
	assert.Equal(t, "rawRequest", contract0.GroupName)
	assert.Equal(t, "RawMessage", contract0.Request.Message.Name)
	assert.Len(t, contract0.Request.Message.Fields, 0)
	assert.Equal(t, desc.KitRawMessage, contract0.Request.Message.Kind)
	assert.Equal(t, "RawMessage", contract0.Responses[0].Message.Name)
	assert.Len(t, contract0.Responses[0].Message.Fields, 0)
	assert.Equal(t, desc.KitRawMessage, contract0.Responses[0].Message.Kind)

	assert.Equal(t, "s2", contract1.Name)
	assert.Equal(t, desc.REST, contract1.Type)
	assert.Equal(t, kit.JSON.Tag(), contract1.Encoding)
	assert.Equal(t, "/multipart/form", contract1.Path)
	assert.Equal(t, "POST", contract1.Method)
	assert.Equal(t, "multipartFormRequest", contract1.GroupName)
	assert.Equal(t, "MultipartFormMessage", contract1.Request.Message.Name)
	assert.Len(t, contract1.Request.Message.Fields, 0)
	assert.Equal(t, desc.KitMultipartFormMessage, contract1.Request.Message.Kind)
	assert.Equal(t, "RawMessage", contract1.Responses[0].Message.Name)
	assert.Len(t, contract1.Responses[0].Message.Fields, 0)
	assert.Equal(t, desc.KitRawMessage, contract1.Responses[0].Message.Kind)
}

type SpecialFields struct {
	T       time.Time             `json:"t"`
	TPtr    *time.Time            `json:"tPtr"`
	TMap    map[string]time.Time  `json:"tMap"`
	TMapPtr map[string]*time.Time `json:"tMapPtr"`
	TArr    []time.Time           `json:"tArr"`
	TArrPtr []*time.Time          `json:"tArrPtr"`
	NUM     utils.Numeric         `json:"num"`
}

func TestTimeFields(t *testing.T) {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("rawRequest").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/raw1", "POST"))).
				In(&SpecialFields{}).
				Out(kit.RawMessage{}),
		)

	pd := desc.ParseService(d)
	contract0 := pd.Contracts[0]
	assert.Len(t, pd.Contracts, 1)

	assert.Equal(t, "s1", contract0.Name)
	assert.Equal(t, desc.REST, contract0.Type)
	assert.Equal(t, kit.JSON.Tag(), contract0.Encoding)
	assert.Equal(t, "/raw1", contract0.Path)
	assert.Equal(t, "POST", contract0.Method)
	assert.Equal(t, "rawRequest", contract0.GroupName)
	assert.Equal(t, "SpecialFields", contract0.Request.Message.Name)
	assert.Len(t, contract0.Request.Message.Fields, 7)
	assert.Equal(t, desc.Object, contract0.Request.Message.Kind)
	assert.Equal(t, "t", contract0.Request.Message.Fields[0].Name)
	assert.Equal(t, reflect.TypeOf(time.Time{}), contract0.Request.Message.Fields[0].Element.RType)
	assert.Equal(t, desc.String, contract0.Request.Message.Fields[0].Element.Kind)
	assert.Equal(t, "tPtr", contract0.Request.Message.Fields[1].Name)
	assert.Equal(t, reflect.TypeOf(&time.Time{}), contract0.Request.Message.Fields[1].Element.RType)
	assert.Equal(t, "tMap", contract0.Request.Message.Fields[2].Name)
	assert.Equal(t, reflect.TypeOf(map[string]time.Time{}), contract0.Request.Message.Fields[2].Element.RType)
	assert.Equal(t, "tMapPtr", contract0.Request.Message.Fields[3].Name)
	assert.Equal(t, reflect.TypeOf(map[string]*time.Time{}), contract0.Request.Message.Fields[3].Element.RType)
	assert.Equal(t, "tArr", contract0.Request.Message.Fields[4].Name)
	assert.Equal(t, reflect.TypeOf([]time.Time{}), contract0.Request.Message.Fields[4].Element.RType)
	assert.Equal(t, "tArrPtr", contract0.Request.Message.Fields[5].Name)
	assert.Equal(t, reflect.TypeOf([]*time.Time{}), contract0.Request.Message.Fields[5].Element.RType)
	assert.Equal(t, "num", contract0.Request.Message.Fields[6].Name)
	assert.Equal(t, reflect.TypeOf(utils.Numeric{}), contract0.Request.Message.Fields[6].Element.RType)
	assert.Equal(t, desc.String, contract0.Request.Message.Fields[6].Element.Kind)
	assert.Equal(t, "RawMessage", contract0.Responses[0].Message.Name)
}
