package mcp

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/x/rkit"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type sayHiInput struct {
	Name  string `json:"name" jsonschema:"required"`
	Phone string `json:"phone" jsonschema:"required"`
}

type sayHiOutput struct {
	Msg string `json:"msg" jsonschema:"required"`
}

func sayHiHandler(ctx *kit.Context) {
	in := ctx.In().GetMsg().(*sayHiInput)
	ctx.In().Reply().SetMsg(&sayHiOutput{Msg: "hi " + in.Name}).Send()
}

func waitForAddr(t *testing.T, b *bundle) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if b.startedAddrIsSet.Load() && b.startedAddr != "" {
			return b.startedAddr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("gateway did not start listening in time")
	return ""
}

func endpointForAddr(t *testing.T, addr string) string {
	t.Helper()
	// addr from net.Listener is already in host:port form, potentially with IPv6.
	u := &url.URL{Scheme: "http", Host: addr}
	return u.String()
}

func TestMCPGateway_StreamableHTTP_ToolLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	gw := MustNew(
		WithName("TestServer"),
		WithTitle("Test Server"),
		WithListener(ln),
	).(*bundle)

	srv := kit.NewServer(
		kit.WithGateway(gw),
		kit.WithServiceBuilder(
			desc.NewService("svc").
				AddContract(
					desc.NewContract().
						In(&sayHiInput{}).
						Out(&sayHiOutput{}).
						AddRoute(desc.Route("sayHi", Selector{
							Name:        "SayHi",
							Title:       "Say Hi",
							Description: "says hi",
						})).
						SetHandler(sayHiHandler),
				),
		),
	)

	srv.Start(ctx)
	defer srv.Shutdown(context.Background())

	addr := waitForAddr(t, gw)
	endpoint := endpointForAddr(t, addr)

	client := sdk.NewClient(&sdk.Implementation{Name: "client"}, nil)
	cs, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if tools == nil || len(tools.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %#v", tools)
	}
	if tools.Tools[0].Name != "SayHi" {
		t.Fatalf("unexpected tool name: %q", tools.Tools[0].Name)
	}

	res, err := cs.CallTool(ctx, &sdk.CallToolParams{
		Name: "SayHi",
		Arguments: map[string]any{
			"name":  "Ehsan",
			"phone": "123",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("unexpected tool error result: %#v", res)
	}
	if res.StructuredContent == nil {
		t.Fatalf("expected StructuredContent to be set")
	}

	t.Log("Tool call successful", rkit.ToJSONStr(res))
}

func TestMCPGateway_StreamableHTTP_InvalidToolArgs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	gw := MustNew(
		WithName("TestServer"),
		WithListener(ln),
	).(*bundle)

	srv := kit.NewServer(
		kit.WithGateway(gw),
		kit.WithServiceBuilder(
			desc.NewService("svc").
				AddContract(
					desc.NewContract().
						In(&sayHiInput{}).
						Out(&sayHiOutput{}).
						AddRoute(desc.Route("sayHi", Selector{
							Name:        "SayHi",
							Title:       "Say Hi",
							Description: "says hi",
						})).
						SetHandler(sayHiHandler),
				),
		),
	)

	srv.Start(ctx)
	defer srv.Shutdown(context.Background())

	addr := waitForAddr(t, gw)
	endpoint := endpointForAddr(t, addr)

	client := sdk.NewClient(&sdk.Implementation{Name: "client"}, nil)
	cs, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// Missing required "phone".
	_, err = cs.CallTool(ctx, &sdk.CallToolParams{
		Name: "SayHi",
		Arguments: map[string]any{
			"name": "Ehsan",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid params error, got nil")
	}
}
