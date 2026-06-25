package kit_test

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
)

type testContract struct {
	handlers  []kit.HandlerFunc
	modifiers []kit.ModifierFunc
}

func (t testContract) ID() string {
	return "test-contract"
}

func (t testContract) RouteSelector() kit.RouteSelector {
	return testSelector{}
}

func (t testContract) EdgeSelector() kit.EdgeSelectorFunc {
	return nil
}

func (t testContract) Encoding() kit.Encoding {
	return kit.JSON
}

func (t testContract) Input() kit.Message {
	return nil
}

func (t testContract) Output() kit.Message {
	return nil
}

func (t testContract) Handlers() []kit.HandlerFunc {
	return t.handlers
}

func (t testContract) Modifiers() []kit.ModifierFunc {
	return t.modifiers
}

type testRESTConn struct {
	*testConn
	statusCode   int
	method       string
	host         string
	requestURI   string
	path         string
	redirectCode int
	redirectURL  string
	query        map[string]string
}

func (t *testRESTConn) GetMethod() string {
	return t.method
}

func (t *testRESTConn) GetHost() string {
	return t.host
}

func (t *testRESTConn) GetRequestURI() string {
	return t.requestURI
}

func (t *testRESTConn) GetPath() string {
	return t.path
}

func (t *testRESTConn) SetStatusCode(code int) {
	t.statusCode = code
}

func (t *testRESTConn) Redirect(code int, url string) {
	t.redirectCode = code
	t.redirectURL = url
}

func (t *testRESTConn) WalkQueryParams(fn func(key string, val string) bool) {
	for k, v := range t.query {
		if !fn(k, v) {
			return
		}
	}
}

func TestContextExecution(t *testing.T) {
	t.Run("should execute handlers in order", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		ctx.SetConn(newTestConn(1, "", false))

		var order []string
		ctr := testContract{
			handlers: []kit.HandlerFunc{
				func(*kit.Context) { order = append(order, "h1") },
				func(*kit.Context) { order = append(order, "h2") },
			},
		}

		ctx.Exec(kit.ExecuteArg{}, ctr)
		assert.Equal(t, []string{"h1", "h2"}, order)
	})

	t.Run("should stop execution when requested", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		ctx.SetConn(newTestConn(1, "", false))

		var order []string
		ctr := testContract{
			handlers: []kit.HandlerFunc{
				func(*kit.Context) { order = append(order, "h1") },
				func(c *kit.Context) {
					order = append(order, "h2")
					c.StopExecution()
				},
				func(*kit.Context) { order = append(order, "h3") },
			},
		}

		ctx.Exec(kit.ExecuteArg{}, ctr)
		assert.Equal(t, []string{"h1", "h2"}, order)
	})
}

func TestContextHeadersAndStatus(t *testing.T) {
	t.Run("should apply preset headers to outgoing envelopes", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		ctx.SetConn(newTestConn(1, "", false))
		ctx.PresetHdr("k1", "v1")

		hdr := map[string]string{"k2": "v2"}
		ctx.PresetHdrMap(hdr)
		hdr["k2"] = "changed"

		out := ctx.Out()
		assert.Equal(t, "v1", out.GetHdr("k1"))
		assert.Equal(t, "v2", out.GetHdr("k2"))
	})

	t.Run("should sync status code with REST connections", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		rc := &testRESTConn{
			testConn: newTestConn(1, "", false),
		}
		ctx.SetConn(rc)
		ctx.SetStatusCode(202)

		assert.Equal(t, 202, ctx.GetStatusCode())
		assert.Equal(t, 202, rc.statusCode)
	})
}

func TestContextModifiersAndErrors(t *testing.T) {
	t.Run("should apply modifiers in LIFO order", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		conn := newTestConn(1, "", false)
		ctx.SetConn(conn)

		var order []string
		ctx.AddModifier(func(e *kit.Envelope) {
			order = append(order, "m1")
			e.SetHdr("m1", "1")
		})
		ctx.AddModifier(func(e *kit.Envelope) {
			order = append(order, "m2")
			e.SetHdr("m2", "2")
		})

		ctx.Out().SetMsg(kit.RawMessage("ok")).Send()
		assert.Equal(t, []string{"m2", "m1"}, order)
		assert.Equal(t, "ok", conn.buf.String())
	})

	t.Run("should record errors", func(t *testing.T) {
		ctx := kit.NewContext(nil)
		assert.False(t, ctx.HasError())

		assert.True(t, ctx.Error(errors.New("boom")))
		assert.True(t, ctx.HasError())
	})
}
