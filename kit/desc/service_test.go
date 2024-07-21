package desc_test

import (
	"fmt"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service - 1", func() {
	d := desc.NewService("sample").
		AddHandler(func(ctx *kit.Context) {}).
		SetEncoding(kit.JSON).
		AddContract(
			desc.NewContract().
				SetName("c1").
				SetEncoding(kit.MSG).
				NamedSelector("s1", newREST(kit.JSON, "/path1", "GET")).
				NamedSelector("s2", newREST(kit.JSON, "/path2", "POST")).
				In(&NestedMessage{}).
				Out(&FlatMessage{}).
				SetHandler(func(ctx *kit.Context) {}),
		)

	svc := d.Build()
	It("should build service", func() {
		Expect(svc).NotTo(BeNil())
		Expect(svc.Contracts()).To(HaveLen(2))
		Expect(svc.Contracts()[0].Encoding()).To(Equal(kit.MSG))
		Expect(svc.Contracts()[0].Input()).To(Equal(&NestedMessage{}))
		Expect(svc.Contracts()[0].Output()).To(Equal(&FlatMessage{}))
		Expect(svc.Contracts()[0].Handlers()).To(HaveLen(2))
	})
})

var _ = Describe("Service - 2", func() {
	d := desc.NewService("sample").
		SetEncoding(kit.JSON).
		SetVersion("v1.0.0").
		SetDescription("some random desc").
		AddError(&testError{code: 400, item: "SOME_BUG"}).
		AddContract(
			desc.NewContract().
				SetName("c1").
				AddError(&testError{code: 401, item: "SOME_ANOTHER_BUG"}).
				NamedSelector("s1", newREST(kit.JSON, "/path1", "GET")).
				NamedSelector("s2", newREST(kit.JSON, "/path2", "POST")).
				In(&NestedMessage{}).
				Out(&FlatMessage{}).
				SetHandler(func(ctx *kit.Context) {}),
		)

	It("should service description have", func() {
		Expect(d.PossibleErrors).To(HaveLen(1))
		Expect(d.Contracts).To(HaveLen(1))
		Expect(d.PossibleErrors[0].Code).To(Equal(400))
		Expect(d.PossibleErrors[0].Item).To(Equal("SOME_BUG"))
		Expect(d.Contracts[0].PossibleErrors).To(HaveLen(1))
		Expect(d.Contracts[0].PossibleErrors[0].Code).To(Equal(401))
		Expect(d.Contracts[0].PossibleErrors[0].Item).To(Equal("SOME_ANOTHER_BUG"))
		Expect(d.Contracts[0].RouteSelectors).To(HaveLen(2))
		Expect(d.Contracts[0].RouteSelectors[0].Selector.GetEncoding()).To(Equal(kit.JSON))
	})
	svc := d.Build()
	It("should build service", func() {
		Expect(svc).NotTo(BeNil())
		Expect(svc.Contracts()).To(HaveLen(2))
		Expect(svc.Contracts()[0].Encoding()).To(Equal(kit.JSON))
		Expect(svc.Contracts()[0].Input()).To(Equal(&NestedMessage{}))
		Expect(svc.Contracts()[0].Output()).To(Equal(&FlatMessage{}))
		Expect(svc.Contracts()[0].Handlers()).To(HaveLen(1))
	})
})

var _ kit.ErrorMessage = (*testError)(nil)

type testError struct {
	code int
	item string
}

func (t testError) GetCode() int {
	return t.code
}

func (t testError) GetItem() string {
	return t.item
}

func (t testError) Error() string {
	return fmt.Sprintf("%d: %s", t.code, t.item)
}
