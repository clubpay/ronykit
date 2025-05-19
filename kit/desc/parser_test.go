package desc_test

import (
	"reflect"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

type FlatMessage struct {
	A string                       `json:"a"`
	B int64                        `json:"b"`
	C map[string]string            `json:"c"`
	D map[string]int64             `json:"d"`
	E map[int64]string             `json:"e"`
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

var _ = Describe("DescParser", func() {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("c1").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/path2", "POST"))).
				In(&NestedMessage{}).
				Out(&FlatMessage{}),
		)

	It("should parse the descriptor", func() {
		pd := desc.ParseService(d)
		contract0 := pd.Contracts[0]
		contract1 := pd.Contracts[1]
		Expect(pd.Contracts).To(HaveLen(2))

		Expect(contract0.Name).To(Equal("s1"))
		Expect(contract0.Type).To(Equal(desc.REST))
		Expect(contract0.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract0.Path).To(Equal("/path1"))
		Expect(contract0.Method).To(Equal("GET"))
		Expect(contract0.GroupName).To(Equal("c1"))
		Expect(contract0.Request.Message.Name).To(Equal("NestedMessage"))
		Expect(contract0.Request.Message.Fields).To(HaveLen(7))
		Expect(contract0.Responses[0].Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Responses[0].Message.Fields).To(HaveLen(8))

		Expect(contract1.Name).To(Equal("s2"))
		Expect(contract1.Type).To(Equal(desc.REST))
		Expect(contract1.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract1.Path).To(Equal("/path2"))
		Expect(contract1.Method).To(Equal("POST"))
		Expect(contract1.GroupName).To(Equal("c1"))

		Expect(contract0.Request.Message.Name).To(Equal("NestedMessage"))
		Expect(contract0.Request.Message.Fields).To(HaveLen(7))
		Expect(contract0.Request.Message.Fields[0].Name).To(Equal("a"))
		Expect(contract0.Request.Message.Fields[0].Element.Kind).To(Equal(desc.String))
		Expect(contract0.Request.Message.Fields[1].Name).To(Equal("b"))
		Expect(contract0.Request.Message.Fields[1].Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.Fields[1].Element.Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Request.Message.Fields[1].Optional).To(BeFalse())
		Expect(contract0.Request.Message.Fields[2].Name).To(Equal("ba"))
		Expect(contract0.Request.Message.Fields[2].Element.Kind).To(Equal(desc.Array))
		Expect(contract0.Request.Message.Fields[2].Element.Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.Fields[2].Element.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Request.Message.Fields[2].Tag.OmitEmpty).To(BeTrue())
		Expect(contract0.Request.Message.Fields[2].Optional).To(BeTrue())
		Expect(contract0.Request.Message.Fields[3].Name).To(Equal("bm"))
		Expect(contract0.Request.Message.Fields[3].Element.Kind).To(Equal(desc.Map))
		Expect(contract0.Request.Message.Fields[3].Element.Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.Fields[3].Element.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Request.Message.Fields[3].Optional).To(BeTrue())
		Expect(contract0.Request.Message.Fields[3].Tag.OmitEmpty).To(BeFalse())
		Expect(contract0.Request.Message.Fields[4].Name).To(Equal("c"))
		Expect(contract0.Request.Message.Fields[4].Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.Fields[4].Element.Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Request.Message.Fields[4].Optional).To(BeTrue())
		Expect(contract0.Request.Message.FieldByName("ba").Element.Kind).To(Equal(desc.Array))
		Expect(contract0.Request.Message.FieldByGoName("BA").Element.Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.FieldByGoName("PA").Element.Kind).To(Equal(desc.Array))
		Expect(contract0.Request.Message.FieldByGoName("PA").Element.Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.FieldByGoName("PA").Element.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Request.Message.FieldByGoName("PMap").Element.Kind).To(Equal(desc.Map))
		Expect(contract0.Request.Message.FieldByGoName("PMap").Element.Element.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.FieldByGoName("PMap").Element.Element.Message.Name).To(Equal("FlatMessage"))

		b := contract0.Request.Message.Fields[1]
		Expect(b.Name).To(Equal("b"))
		Expect(b.Element.Kind).To(Equal(desc.Object))
		Expect(b.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(b.Element.Message.Fields).To(HaveLen(8))
		Expect(b.Element.Message.Fields[0].Name).To(Equal("a"))
		Expect(b.Element.Message.Fields[0].Element.Kind).To(Equal(desc.String))

		ba := contract0.Request.Message.Fields[2]
		Expect(ba.Name).To(Equal("ba"))
		Expect(ba.Element.Kind).To(Equal(desc.Array))
		Expect(ba.Element.Message).To(BeNil())
		Expect(ba.Element.Element.Kind).To(Equal(desc.Object))
		Expect(ba.Element.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(ba.Element.Element.Message.Fields).To(HaveLen(8))
		Expect(ba.Element.Element.Message.Fields[0].Name).To(Equal("a"))
		Expect(ba.Element.Element.Message.Fields[0].Element.Kind).To(Equal(desc.String))

		g := contract0.Responses[0].Message.Fields[5]
		Expect(g.Name).To(Equal("g"))
		Expect(g.Element.Kind).To(Equal(desc.Array))
		Expect(g.Element.Element.Kind).To(Equal(desc.Array))
		Expect(g.Element.Element.Element.Kind).To(Equal(desc.String))
		Expect(g.Element.Element.Element.Message).To(BeNil())
		Expect(g.Element.Element.Message).To(BeNil())

		m := contract0.Responses[0].Message.Fields[6]
		Expect(m.Name).To(Equal("m"))
		Expect(m.Element.Kind).To(Equal(desc.Map))
		Expect(m.Element.Element.Kind).To(Equal(desc.Map))
		Expect(m.Element.Element.Message).To(BeNil())
		Expect(m.Element.Element.Element.Kind).To(Equal(desc.String))
		Expect(m.Element.Element.Element.Message).To(BeNil())

		f := contract0.Responses[0].Message.Fields[2]
		Expect(f.Name).To(Equal("c"))
		Expect(f.Element.Kind).To(Equal(desc.Map))
		Expect(f.Element.Element.Kind).To(Equal(desc.String))
		Expect(f.Element.Message).To(BeNil())
		Expect(f.Optional).To(BeTrue())
		Expect(f.Element.Message).To(BeNil())
		Expect(f.Element.Type).To(Equal("map[string]string"))

		f3 := contract0.Responses[0].Message.Fields[3]
		Expect(f3.Name).To(Equal("d"))
		Expect(f3.Element.Kind).To(Equal(desc.Map))
		Expect(f3.Element.Element.Kind).To(Equal(desc.Integer))
		Expect(f3.Element.Message).To(BeNil())
		Expect(f3.Optional).To(BeTrue())
		Expect(f3.Element.Message).To(BeNil())
		Expect(f3.Element.Type).To(Equal("map[string]int64"))
	})
})

var _ = Describe("ParseMessage.JSON()", func() {
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

	It("Parse Service", func() {
		ps := desc.ParseService(d)
		Expect(ps.Messages()).To(HaveLen(2))
		Expect(ps.Contracts).To(HaveLen(2))
		Expect(ps.Contracts[0].Type).To(Equal(desc.REST))
		Expect(ps.Contracts[0].Request.Headers).To(HaveLen(2))
		Expect(ps.Contracts[0].Request.Headers[0].Required).To(BeTrue())
		Expect(ps.Contracts[0].Request.Headers[0].Name).To(Equal("hdr1"))
		Expect(ps.Contracts[0].Request.Headers[1].Required).To(BeFalse())
		Expect(ps.Contracts[0].Request.Headers[1].Name).To(Equal("optionalHdr1"))
	})
})

var _ = Describe("RawMessage and MultipartForm", func() {
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

	It("should parse the descriptor", func() {
		pd := desc.ParseService(d)
		contract0 := pd.Contracts[0]
		contract1 := pd.Contracts[1]
		Expect(pd.Contracts).To(HaveLen(2))

		Expect(contract0.Name).To(Equal("s1"))
		Expect(contract0.Type).To(Equal(desc.REST))
		Expect(contract0.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract0.Path).To(Equal("/raw1"))
		Expect(contract0.Method).To(Equal("POST"))
		Expect(contract0.GroupName).To(Equal("rawRequest"))
		Expect(contract0.Request.Message.Name).To(Equal("RawMessage"))
		Expect(contract0.Request.Message.Fields).To(HaveLen(0))
		Expect(contract0.Request.Message.Kind).To(Equal(desc.KitRawMessage))
		Expect(contract0.Responses[0].Message.Name).To(Equal("RawMessage"))
		Expect(contract0.Responses[0].Message.Fields).To(HaveLen(0))
		Expect(contract0.Responses[0].Message.Kind).To(Equal(desc.KitRawMessage))

		Expect(contract1.Name).To(Equal("s2"))
		Expect(contract1.Type).To(Equal(desc.REST))
		Expect(contract1.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract1.Path).To(Equal("/multipart/form"))
		Expect(contract1.Method).To(Equal("POST"))
		Expect(contract1.GroupName).To(Equal("multipartFormRequest"))
		Expect(contract1.Request.Message.Name).To(Equal("MultipartFormMessage"))
		Expect(contract1.Request.Message.Fields).To(HaveLen(0))
		Expect(contract1.Request.Message.Kind).To(Equal(desc.KitMultipartFormMessage))
		Expect(contract1.Responses[0].Message.Name).To(Equal("RawMessage"))
		Expect(contract1.Responses[0].Message.Fields).To(HaveLen(0))
		Expect(contract1.Responses[0].Message.Kind).To(Equal(desc.KitRawMessage))
	})
})

type SpecialFields struct {
	T       time.Time             `json:"t"`
	TPtr    *time.Time            `json:"tPtr"`
	TMap    map[string]time.Time  `json:"tMap"`
	TMapPtr map[string]*time.Time `json:"tMapPtr"`
	TArr    []time.Time           `json:"tArr"`
	TArrPtr []*time.Time          `json:"tArrPtr"`
}

var _ = Describe("Time Fields", func() {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("rawRequest").
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/raw1", "POST"))).
				In(&SpecialFields{}).
				Out(kit.RawMessage{}),
		)

	It("should parse the descriptor", func() {
		pd := desc.ParseService(d)
		contract0 := pd.Contracts[0]
		Expect(pd.Contracts).To(HaveLen(1))

		Expect(contract0.Name).To(Equal("s1"))
		Expect(contract0.Type).To(Equal(desc.REST))
		Expect(contract0.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract0.Path).To(Equal("/raw1"))
		Expect(contract0.Method).To(Equal("POST"))
		Expect(contract0.GroupName).To(Equal("rawRequest"))
		Expect(contract0.Request.Message.Name).To(Equal("SpecialFields"))
		Expect(contract0.Request.Message.Fields).To(HaveLen(6))
		Expect(contract0.Request.Message.Kind).To(Equal(desc.Object))
		Expect(contract0.Request.Message.Fields[0].Name).To(Equal("t"))
		Expect(contract0.Request.Message.Fields[0].Element.RType).To(Equal(reflect.TypeOf(time.Time{})))
		Expect(contract0.Request.Message.Fields[0].Element.Kind).To(Equal(desc.String))
		Expect(contract0.Request.Message.Fields[1].Name).To(Equal("tPtr"))
		Expect(contract0.Request.Message.Fields[1].Element.RType).To(Equal(reflect.TypeOf(&time.Time{})))
		Expect(contract0.Request.Message.Fields[2].Name).To(Equal("tMap"))
		Expect(contract0.Request.Message.Fields[2].Element.RType).To(Equal(reflect.TypeOf(map[string]time.Time{})))
		Expect(contract0.Request.Message.Fields[3].Name).To(Equal("tMapPtr"))
		Expect(contract0.Request.Message.Fields[3].Element.RType).To(Equal(reflect.TypeOf(map[string]*time.Time{})))
		Expect(contract0.Request.Message.Fields[4].Name).To(Equal("tArr"))
		Expect(contract0.Request.Message.Fields[4].Element.RType).To(Equal(reflect.TypeOf([]time.Time{})))
		Expect(contract0.Request.Message.Fields[5].Name).To(Equal("tArrPtr"))
		Expect(contract0.Request.Message.Fields[5].Element.RType).To(Equal(reflect.TypeOf([]*time.Time{})))
		Expect(contract0.Responses[0].Message.Name).To(Equal("RawMessage"))
	})
})
