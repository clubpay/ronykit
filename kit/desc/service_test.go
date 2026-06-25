package desc_test

import (
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/stretchr/testify/assert"
)

func TestService1(t *testing.T) {
	d := desc.NewService("sample").
		AddHandler(func(ctx *kit.Context) {}).
		SetEncoding(kit.JSON).
		AddContract(
			desc.NewContract().
				SetName("c1").
				SetEncoding(kit.MSG).
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/path2", "POST"))).
				In(&NestedMessage{}).
				Out(&FlatMessage{}).
				SetHandler(func(ctx *kit.Context) {}),
		)

	svc := d.Build()
	assert.NotNil(t, svc)
	assert.Len(t, svc.Contracts(), 2)
	assert.Equal(t, kit.MSG, svc.Contracts()[0].Encoding())
	assert.Equal(t, &NestedMessage{}, svc.Contracts()[0].Input())
	assert.Equal(t, &FlatMessage{}, svc.Contracts()[0].Output())
	assert.Len(t, svc.Contracts()[0].Handlers(), 2)
}

func TestService2(t *testing.T) {
	d := desc.NewService("sample").
		SetEncoding(kit.JSON).
		SetVersion("v1.0.0").
		SetDescription("some random desc").
		AddError(&testError{code: 400, item: "SOME_BUG"}).
		AddContract(
			desc.NewContract().
				SetName("c1").
				AddError(&testError{code: 401, item: "SOME_ANOTHER_BUG"}).
				AddRoute(desc.Route("s1", newREST(kit.JSON, "/path1", "GET"))).
				AddRoute(desc.Route("s2", newREST(kit.JSON, "/path2", "POST"))).
				In(&NestedMessage{}).
				Out(&FlatMessage{}).
				SetHandler(func(ctx *kit.Context) {}),
		)

	assert.Len(t, d.PossibleErrors, 1)
	assert.Len(t, d.Contracts, 1)
	assert.Equal(t, 400, d.PossibleErrors[0].Code)
	assert.Equal(t, "SOME_BUG", d.PossibleErrors[0].Item)
	assert.Len(t, d.Contracts[0].PossibleErrors, 1)
	assert.Equal(t, 401, d.Contracts[0].PossibleErrors[0].Code)
	assert.Equal(t, "SOME_ANOTHER_BUG", d.Contracts[0].PossibleErrors[0].Item)
	assert.Len(t, d.Contracts[0].RouteSelectors, 2)
	assert.Equal(t, kit.JSON, d.Contracts[0].RouteSelectors[0].Selector.GetEncoding())

	svc := d.Build()
	assert.NotNil(t, svc)
	assert.Len(t, svc.Contracts(), 2)
	assert.Equal(t, kit.JSON, svc.Contracts()[0].Encoding())
	assert.Equal(t, &NestedMessage{}, svc.Contracts()[0].Input())
	assert.Equal(t, &FlatMessage{}, svc.Contracts()[0].Output())
	assert.Len(t, svc.Contracts()[0].Handlers(), 1)
}

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
