package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/x/rkit"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
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

	addr string
	ln   net.Listener

	httpSrv *http.Server

	serverOpts       mcp.ServerOptions
	streamableOpts   mcp.StreamableHTTPOptions
	serverConfigFns  []func(*mcp.Server)
	nextConnID       atomic.Uint64
	startedAddr      string
	startedAddrIsSet atomic.Bool
}

var _ kit.Gateway = (*bundle)(nil)

func New(opts ...Option) (kit.Gateway, error) {
	b := &bundle{
		addr: ":8080",
	}

	for _, opt := range opts {
		opt(b)
	}

	// Default to an EventStore so stream replay/resumption is supported out of the box.
	// This is memory-backed and bounded by the SDK default (10MiB).
	if b.streamableOpts.EventStore == nil && !b.streamableOpts.Stateless {
		b.streamableOpts.EventStore = mcp.NewMemoryEventStore(nil)
	}

	optsCopy := b.serverOpts
	// If user didn't specify Capabilities, we still want default logging capability
	// (the SDK defaults to that when Capabilities is nil).
	optsCopy.Instructions = b.instructions

	b.srv = mcp.NewServer(
		&mcp.Implementation{
			Name:       b.name,
			Title:      b.title,
			WebsiteURL: b.websiteURL,
		},
		&optsCopy,
	)
	for _, fn := range b.serverConfigFns {
		fn(b.srv)
	}

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
	_ = cfg // reserved for future use (reuseport is handled by providing a listener)

	if b.httpSrv == nil {
		b.httpSrv = &http.Server{}
	}

	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server { return b.srv },
		&b.streamableOpts,
	)
	b.httpSrv.Handler = handler

	if b.ln == nil {
		ln, err := net.Listen("tcp", b.addr)
		if err != nil {
			return err
		}
		b.ln = ln
	}

	if b.startedAddrIsSet.CompareAndSwap(false, true) {
		b.startedAddr = b.ln.Addr().String()
	}

	go func() {
		// Serve returns http.ErrServerClosed on graceful shutdown.
		err := b.httpSrv.Serve(b.ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// No logger in kit.Gateway; errors surface via tests/observability.
			_ = err
		}
	}()

	go func() {
		<-ctx.Done()
		_ = b.Shutdown(context.Background())
	}()

	return nil
}

func (b *bundle) Shutdown(_ context.Context) error {
	if b.httpSrv == nil {
		return nil
	}
	return b.httpSrv.Close()
}

func (b *bundle) Register(
	serviceName, contractID string,
	enc kit.Encoding,
	sel kit.RouteSelector,
	input, output kit.Message,
) {
	inputSchema := rkit.Must(jsonschema.ForType(reflect.Indirect(reflect.ValueOf(input)).Type(), &jsonschema.ForOptions{}))

	outputSchema := rkit.Must(
		jsonschema.ForType(reflect.Indirect(reflect.ValueOf(output)).Type(), &jsonschema.ForOptions{}),
	)
	if inputSchema.Type != "object" || outputSchema.Type != "object" {
		return
	}

	inputResolved, err := inputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		return
	}
	outputResolved, err := outputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
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
			inResolved:  inputResolved,
			outResolved: outputResolved,
		}),
	)
}

type routeData struct {
	sel         kit.RouteSelector
	enc         kit.Encoding
	factory     kit.MessageFactoryFunc
	serviceName string
	contractID  string

	inResolved  *jsonschema.Resolved
	outResolved *jsonschema.Resolved
}

func (b *bundle) getHandler(rd routeData) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Validate and apply defaults to arguments according to the inferred input schema,
		// mirroring the MCP SDK behavior.
		args := req.Params.Arguments
		if rd.inResolved != nil {
			var v map[string]any
			if len(args) > 0 {
				if err := json.Unmarshal(args, &v); err != nil {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error()}
				}
			} else {
				v = map[string]any{}
			}

			if err := rd.inResolved.ApplyDefaults(&v); err != nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error()}
			}
			if err := rd.inResolved.Validate(&v); err != nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error()}
			}

			bb, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("marshal validated args: %w", err)
			}
			req2 := *req
			req2.Params = &mcp.CallToolParamsRaw{
				Meta:      req.Params.Meta,
				Name:      req.Params.Name,
				Arguments: json.RawMessage(bb),
			}
			req = &req2
		}

		conn := &toolConn{
			id:  b.nextConnID.Add(1),
			rd:  rd,
			req: req,
			res: &mcp.CallToolResult{},
		}

		b.d.OnOpen(conn)
		defer b.d.OnClose(conn.ConnID())

		b.d.OnMessage(conn, req.Params.Arguments)

		// Ensure we always return a non-nil result shape (avoid null content).
		if conn.res == nil {
			conn.res = &mcp.CallToolResult{Content: []mcp.Content{}}
		} else if conn.res.Content == nil {
			conn.res.Content = []mcp.Content{}
		}

		return conn.res, nil
	}
}

func (b *bundle) Subscribe(d kit.GatewayDelegate) {
	b.d = d
}

func (b *bundle) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	conn := ctx.Conn().(*toolConn)

	m := conn.rd.factory()

	// At this point input has already been validated by getHandler.
	err := kit.UnmarshalMessage(in, m)
	if err != nil {
		return kit.ExecuteArg{}, err
	}

	env := ctx.In().SetMsg(m)
	for k, v := range conn.req.GetParams().GetMeta() {
		env.SetHdr(k, cast.ToString(v))
	}

	conn.res = &mcp.CallToolResult{
		Content:           []mcp.Content{},
		StructuredContent: nil,
		IsError:           false,
	}

	return kit.ExecuteArg{
		ServiceName: conn.rd.serviceName,
		ContractID:  conn.rd.contractID,
		Route:       conn.rd.sel.Query(queryName).(string),
	}, nil
}
