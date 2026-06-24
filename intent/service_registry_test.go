package intent_test

import (
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/rony"
)

func TestMapServiceRegistryRegisterAndGet(t *testing.T) {
	reg := intent.NewMapServiceRegistry()

	desc := intent.ServiceDescriptor{
		Name: "chat",
		Mount: func(m intent.EndpointMount) error {
			intent.Setup(m, "ChatService", rony.EmptyState(), rony.WithUnary(
				func(ctx *rony.UnaryCtx[rony.EMPTY, rony.NOP], _ string) (*string, error) {
					out := "ok"

					return &out, nil
				},
				rony.GET("/ping"),
			))

			return nil
		},
	}

	if err := reg.Register(desc); err != nil {
		t.Fatal(err)
	}

	got, ok := reg.Get("chat")
	if !ok {
		t.Fatal("expected service to be registered")
	}
	if got.Name != "chat" {
		t.Fatalf("got name %q", got.Name)
	}
	if len(reg.All()) != 1 {
		t.Fatalf("expected 1 service, got %d", len(reg.All()))
	}
}

func TestAgentMountsServiceDescriptor(t *testing.T) {
	agent := intent.New(
		intent.WithName("test-agent"),
		intent.WithService(intent.ServiceDescriptor{
			Name: "chat",
			Mount: func(m intent.EndpointMount) error {
				intent.Setup(m, "ChatService", rony.EmptyState(), rony.WithUnary(
					func(ctx *rony.UnaryCtx[rony.EMPTY, rony.NOP], _ string) (*string, error) {
						out := "ok"

						return &out, nil
					},
					rony.GET("/ping"),
				))

				return nil
			},
		}),
	)

	descs := agent.ExportDesc()
	if len(descs) != 1 {
		t.Fatalf("expected 1 exported service, got %d", len(descs))
	}
}
