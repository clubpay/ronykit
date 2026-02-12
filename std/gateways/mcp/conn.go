package mcp

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolConn struct {
	id  uint64
	req *mcp.CallToolRequest
	res *mcp.CallToolResult
	rd  routeData
}

var _ kit.Conn = (*toolConn)(nil)

func (c *toolConn) ConnID() uint64 {
	return c.id
}

func (c *toolConn) ClientIP() string {
	return ""
}

func (c *toolConn) WriteEnvelope(e *kit.Envelope) error {
	msg := e.GetMsg()

	c.res = &mcp.CallToolResult{
		StructuredContent: msg,
		IsError:           false,
	}

	meta := map[string]any{}
	e.WalkHdr(func(key string, val string) bool {
		meta[key] = val

		return true
	})
	c.res.Meta.SetMeta(meta)

	return nil
}

func (c *toolConn) Stream() bool {
	//TODO implement me
	return true
}

func (c *toolConn) Walk(fn func(key string, val string) bool) {
	for k, v := range c.req.GetParams().GetMeta() {
		if vs, ok := v.(string); ok {
			if !fn(k, vs) {
				return
			}
		}
	}
}

func (c *toolConn) Get(key string) string {
	meta := c.req.GetParams().GetMeta()
	if meta == nil {
		return ""
	}

	return rkit.TryCast[string](meta[key])
}

func (c *toolConn) Set(key string, val string) {
	meta := c.res.Meta.GetMeta()
	meta[key] = val
	c.res.Meta.SetMeta(meta)
}
