package ronykit

var (
	NewEnvelope = newEnvelope
	NewContext  = newContext
)

func (ctx *Context) SetConn(c Conn) {
	ctx.conn = c
	ctx.in = newEnvelope(ctx, c, false)
}

func (ctx *Context) Exec(arg ExecuteArg, c Contract) {
	ctx.execute(arg, c)
}
