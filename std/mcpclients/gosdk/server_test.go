package gosdk_test

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	gw "github.com/clubpay/ronykit/std/gateways/mcp"
	"github.com/clubpay/ronykit/std/mcpclients/gosdk"
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

func startTestMCPServer(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	gateway, err := gw.New(
		gw.WithName("TestServer"),
		gw.WithListener(ln),
	)
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}

	srv := kit.NewServer(
		kit.WithGateway(gateway),
		kit.WithServiceBuilder(
			desc.NewService("svc").
				AddContract(
					desc.NewContract().
						In(&sayHiInput{}).
						Out(&sayHiOutput{}).
						AddRoute(desc.Route("sayHi", gw.Selector{
							Name:        "SayHi",
							Description: "say hi",
						})).
						SetHandler(sayHiHandler),
				),
		),
	)

	srv.Start(ctx)
	t.Cleanup(func() { srv.Shutdown(context.Background()) })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gateway != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return (&url.URL{Scheme: "http", Host: ln.Addr().String()}).String()
}

func TestClientAdapter_CallTool(t *testing.T) {
	endpoint := startTestMCPServer(t)

	factory := gosdk.NewFactory("test-client")
	server, err := factory.NewServer(context.Background(), intent.MCPServerConfig{
		Name:      "test",
		Transport: intent.MCPTransportStreamableHTTP,
		URL:       endpoint,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	result, err := server.CallTool(context.Background(), "SayHi", []byte(`{"name":"Ehsan","phone":"123"}`))
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %#v", result)
	}
	if len(result.Content) == 0 {
		t.Fatalf("expected content blocks")
	}
}

func TestRegistryConnectAll(t *testing.T) {
	endpoint := startTestMCPServer(t)

	factory := gosdk.NewFactory("test-client")
	reg := gosdk.NewRegistry(factory, intent.MCPServerConfig{
		Name:      "test",
		Transport: intent.MCPTransportStreamableHTTP,
		URL:       endpoint,
	})

	err := reg.ConnectAll(context.Background())
	if err != nil {
		t.Fatalf("connect all: %v", err)
	}
	t.Cleanup(func() { _ = reg.CloseAll() })

	srv, ok := reg.Get("test")
	if !ok || srv == nil {
		t.Fatalf("expected connected server")
	}

	tools, err := srv.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "SayHi" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
}
