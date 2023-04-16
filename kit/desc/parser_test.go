package desc_test

import (
	"fmt"

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

func (d dummyRESTSelector) Query(_ string) interface{} {
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
}

type NestedMessage struct {
	A  string                 `json:"a"`
	B  FlatMessage            `json:"b"`
	BA []FlatMessage          `json:"ba"`
	BM map[string]FlatMessage `json:"bm"`
}

var _ = Describe("DescParser", func() {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("c1").
				NamedSelector("s1", newREST(kit.JSON, "/path1", "GET")).
				NamedSelector("s2", newREST(kit.JSON, "/path2", "POST")).
				In(&NestedMessage{}).
				Out(&FlatMessage{}),
		)

	It("should parse the description", func() {
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
		Expect(contract0.Request.Message.Fields).To(HaveLen(4))
		Expect(contract0.Responses[0].Message.Name).To(Equal("FlatMessage"))
		Expect(contract0.Responses[0].Message.Fields).To(HaveLen(6))

		Expect(contract1.Name).To(Equal("s2"))
		Expect(contract1.Type).To(Equal(desc.REST))
		Expect(contract1.Encoding).To(Equal(kit.JSON.Tag()))
		Expect(contract1.Path).To(Equal("/path2"))
		Expect(contract1.Method).To(Equal("POST"))
		Expect(contract1.GroupName).To(Equal("c1"))

		Expect(contract0.Request.Message.Name).To(Equal("NestedMessage"))
		Expect(contract0.Request.Message.Fields).To(HaveLen(4))
		Expect(contract0.Request.Message.Fields[0].Name).To(Equal("a"))
		Expect(contract0.Request.Message.Fields[0].Kind).To(Equal(desc.String))

		b := contract0.Request.Message.Fields[1]
		Expect(b.Name).To(Equal("b"))
		Expect(b.Kind).To(Equal(desc.Object))
		Expect(b.Message.Name).To(Equal("FlatMessage"))
		Expect(b.Message.Fields).To(HaveLen(6))
		Expect(b.Message.Fields[0].Name).To(Equal("a"))
		Expect(b.Message.Fields[0].Kind).To(Equal(desc.String))

		ba := contract0.Request.Message.Fields[2]
		Expect(ba.Name).To(Equal("ba"))
		Expect(ba.Kind).To(Equal(desc.Array))
		Expect(ba.Message).To(BeNil())
		Expect(ba.Element.Kind).To(Equal(desc.Object))
		Expect(ba.Element.Message.Name).To(Equal("FlatMessage"))
		Expect(ba.Element.Message.Fields).To(HaveLen(6))
		Expect(ba.Element.Message.Fields[0].Name).To(Equal("a"))
		Expect(ba.Element.Message.Fields[0].Kind).To(Equal(desc.String))

		g := contract0.Responses[0].Message.Fields[4]
		Expect(g.Name).To(Equal("g"))
		Expect(g.Kind).To(Equal(desc.Array))
		Expect(g.Element.Kind).To(Equal(desc.Array))
		Expect(g.Element.Element.Kind).To(Equal(desc.String))
		Expect(g.Element.Element.Message).To(BeNil())
		Expect(g.Element.Message).To(BeNil())

		m := contract0.Responses[0].Message.Fields[5]
		Expect(m.Name).To(Equal("m"))
		Expect(m.Kind).To(Equal(desc.Map))
		Expect(m.Element.Kind).To(Equal(desc.Map))
		Expect(m.Element.Message).To(BeNil())
		Expect(m.Element.Element.Kind).To(Equal(desc.String))
		Expect(m.Element.Element.Message).To(BeNil())
	})
})

var _ = Describe("ParseMessage.JSON()", func() {
	d := desc.NewService("sample").
		AddContract(
			desc.NewContract().
				SetName("c1").
				NamedSelector("s1", newREST(kit.JSON, "/path1", "GET")).
				NamedSelector("s2", newREST(kit.JSON, "/path2", "POST")).
				In(&NestedMessage{}).
				Out(&FlatMessage{}),
		)

	ps := desc.ParseService(d)
	fmt.Println(ps.Contracts[0].Request.Message.JSON())
})
