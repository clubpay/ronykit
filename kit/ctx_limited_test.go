package kit_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
)

func TestCtxLimited(t *testing.T) {
	ctx := kit.NewContext(nil)
	conn := newTestConn(100, "", true)
	ctx.SetConn(conn)

	limitCtx := ctx.Limited()
	assert.Equal(t, conn.ConnID(), limitCtx.Conn().ConnID())
	assert.Equal(t, ctx.ServiceName(), limitCtx.ServiceName())
	limitCtx.In().SetHdr("k1", "v1")
	assert.Equal(t, "v1", ctx.In().GetHdr("k1"))
}
