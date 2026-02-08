package desc_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

func TestContractAndServiceHelpers(t *testing.T) {
	opt := desc.OptionalHeader("x-opt")
	if opt.Name != "x-opt" || opt.Required {
		t.Fatalf("unexpected optional header: %+v", opt)
	}
	req := desc.RequiredHeader("x-req")
	if req.Name != "x-req" || !req.Required {
		t.Fatalf("unexpected required header: %+v", req)
	}

	route := desc.Route("r1", newREST(kit.JSON, "/path", "GET"))
	if route.Deprecated {
		t.Fatalf("expected route to be not deprecated")
	}
	if !route.Deprecate().Deprecated {
		t.Fatalf("expected route to be deprecated")
	}

	c := desc.NewContract()
	if c.Encoding != kit.JSON {
		t.Fatalf("expected default encoding to be JSON")
	}
	c.SetName("c1").
		SetEncoding(kit.MSG).
		SetInputHeader(opt, req).
		In(&NestedMessage{}, desc.WithField("a", desc.FieldMeta{Optional: true})).
		Out(&FlatMessage{}).
		AddRoute(route).
		AddSelector(newREST(kit.JSON, "/s1", "GET")).
		AddNamedSelector("named", newREST(kit.JSON, "/s2", "POST")).
		SetCoordinator(func(*kit.LimitedContext) (string, error) { return "edge-1", nil }).
		AddModifier(func(*kit.Envelope) {}).
		AddWrapper(kit.ContractWrapperFunc(func(c kit.Contract) kit.Contract { return c })).
		AddHandler(func(*kit.Context) {}).
		SetHandler(func(*kit.Context) {})

	if c.Name != "c1" || len(c.RouteSelectors) != 3 {
		t.Fatalf("unexpected contract state: %+v", c)
	}
	if len(c.InputMeta.Fields) != 1 || c.InputMeta.Fields["a"].Optional != true {
		t.Fatalf("unexpected input meta: %+v", c.InputMeta)
	}

	svc := desc.NewService("svc").
		SetEncoding(kit.MSG).
		AddContract(c)

	alias := desc.NewContract()
	alias.Selector(newREST(kit.JSON, "/alias", "GET"))
	alias.NamedSelector("alias", newREST(kit.JSON, "/alias2", "POST"))
	alias.Coordinator(func(*kit.LimitedContext) (string, error) { return "edge-2", nil })

	undef := desc.NewContract().
		SetName("undef").
		In(&NestedMessage{}).
		Out(&FlatMessage{}).
		AddRoute(desc.Route("r2", newREST(kit.JSON, "/undef", "POST")))
	undef.Encoding = kit.Undefined
	svc.AddContract(undef)

	form := desc.NewContract().
		SetName("upload").
		SetInput(kit.MultipartFormMessage{}).
		AddRoute(desc.Route("r3", newREST(kit.JSON, "/upload", "POST")))
	form.Encoding = kit.Undefined
	svc.AddContract(form)

	descs := desc.ToDesc(svc)
	if len(descs) != 1 || descs[0].Desc().Name != "svc" {
		t.Fatalf("unexpected service desc list: %v", descs)
	}

	built := desc.BuildService(descs[0])
	if built.Name() != "svc" || len(built.Contracts()) != 5 {
		t.Fatalf("unexpected built service: %s, %d", built.Name(), len(built.Contracts()))
	}
	if built.Contracts()[0].Encoding() != kit.MSG {
		t.Fatalf("unexpected contract encoding: %v", built.Contracts()[0].Encoding())
	}
	if built.Contracts()[3].Encoding() != kit.MSG {
		t.Fatalf("unexpected undefined contract encoding: %v", built.Contracts()[3].Encoding())
	}
	if built.Contracts()[4].Encoding() != kit.MultipartForm {
		t.Fatalf("unexpected multipart contract encoding: %v", built.Contracts()[4].Encoding())
	}

	first := built.Contracts()[0]
	if first.ID() == "" || first.RouteSelector() == nil || first.EdgeSelector() == nil {
		t.Fatalf("unexpected contract accessors: %q, %v, %v", first.ID(), first.RouteSelector(), first.EdgeSelector())
	}
	if len(first.Modifiers()) != 1 {
		t.Fatalf("unexpected modifiers length: %d", len(first.Modifiers()))
	}
}

func TestServiceBuildDuplicateContractName(t *testing.T) {
	svc := desc.NewService("svc").
		AddContract(
			desc.NewContract().
				SetName("dup").
				AddRoute(desc.Route("r1", newREST(kit.JSON, "/one", "GET"))),
			desc.NewContract().
				SetName("dup").
				AddRoute(desc.Route("r2", newREST(kit.JSON, "/two", "POST"))),
		)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate contract name")
		}
	}()
	_ = svc.Build()
}
