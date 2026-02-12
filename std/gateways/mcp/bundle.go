package mcp

import (
	"context"
	"net/http"
	"reflect"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cast"
)

type bundle struct {
	srv *mcp.Server
	d   kit.GatewayDelegate

	name         string
	title        string
	websiteURL   string
	instructions string
}

var _ kit.Gateway = (*bundle)(nil)

func New(opts ...Option) (kit.Gateway, error) {
	b := &bundle{}

	for _, opt := range opts {
		opt(b)
	}

	b.srv = mcp.NewServer(
		&mcp.Implementation{
			Name:       b.name,
			Title:      b.title,
			WebsiteURL: b.websiteURL,
		},
		&mcp.ServerOptions{
			Instructions: b.instructions,
		},
	)

	return b, nil
}

func MustNew(opts ...Option) kit.Gateway {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (b *bundle) Start(ctx context.Context, cfg kit.GatewayStartConfig) error {
	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server { return b.srv },
		nil,
	)

	go func() {
		_ = http.ListenAndServe(":8080", handler)
	}()

	return nil
}

func (b *bundle) Shutdown(_ context.Context) error {
	return nil
}

func (b *bundle) Register(
	serviceName, contractID string,
	enc kit.Encoding,
	sel kit.RouteSelector,
	input, output kit.Message,
) {
	inputSchema := rkit.Must(jsonschema.ForType(reflect.Indirect(reflect.ValueOf(input)).Type(), &jsonschema.ForOptions{}))
	outputSchema := rkit.Must(jsonschema.ForType(reflect.Indirect(reflect.ValueOf(output)).Type(), &jsonschema.ForOptions{}))
	if inputSchema.Type != "object" || outputSchema.Type != "object" {
		return
	}

	b.srv.AddTool(
		&mcp.Tool{
			Meta: nil,
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: sel.Query(queryDestructive).(*bool),
				IdempotentHint:  sel.Query(queryIdempotent).(bool),
				OpenWorldHint:   sel.Query(queryOpenWorld).(*bool),
				ReadOnlyHint:    sel.Query(queryReadOnly).(bool),
				Title:           sel.Query(queryTitle).(string),
			},
			Description:  sel.Query(queryDesc).(string),
			InputSchema:  inputSchema,
			Name:         sel.Query(queryName).(string),
			OutputSchema: outputSchema,
			Title:        sel.Query(queryTitle).(string),
		},
		b.getHandler(routeData{
			sel:         sel,
			enc:         enc,
			serviceName: serviceName,
			contractID:  contractID,
			factory:     kit.CreateMessageFactory(input),
		}),
	)
}

type routeData struct {
	sel         kit.RouteSelector
	enc         kit.Encoding
	factory     kit.MessageFactoryFunc
	serviceName string
	contractID  string
}

func (b *bundle) getHandler(rd routeData) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		conn := &toolConn{
			rd:  rd,
			req: req,
			res: &mcp.CallToolResult{},
		}

		b.d.OnOpen(conn)
		b.d.OnMessage(conn, req.Params.Arguments)
		b.d.OnClose(conn.ConnID())

		return conn.res, nil
	}
}

func (b *bundle) Subscribe(d kit.GatewayDelegate) {
	b.d = d
}

func (b *bundle) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	conn := ctx.Conn().(*toolConn)

	m := conn.rd.factory()
	err := kit.UnmarshalMessage(in, m)
	if err != nil {
		return kit.ExecuteArg{}, err
	}

	env := ctx.In().SetMsg(m)
	for k, v := range conn.req.GetParams().GetMeta() {
		env.SetHdr(k, cast.ToString(v))
	}

	conn.res = &mcp.CallToolResult{
		Content:           nil,
		StructuredContent: nil,
		IsError:           false,
	}

	return kit.ExecuteArg{
		ServiceName: conn.rd.serviceName,
		ContractID:  conn.rd.contractID,
		Route:       conn.rd.sel.Query(queryName).(string),
	}, nil
}
