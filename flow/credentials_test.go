package flow

import (
	"context"
	"crypto/tls"
	"testing"
)

func TestCredentialsHelpers(t *testing.T) {
	static := NewAPIKeyStaticCredentials("test-key")
	if static == nil {
		t.Fatal("expected static API key credentials")
	}

	dynamic := NewAPIKeyDynamicCredentials(func(context.Context) (string, error) {
		return "rotated-key", nil
	})
	if dynamic == nil {
		t.Fatal("expected dynamic API key credentials")
	}

	mtls := NewMTLSCredentials(tls.Certificate{})
	if mtls == nil {
		t.Fatal("expected mTLS credentials")
	}

	cfg := BackendConfig{Credentials: static}
	if cfg.Credentials == nil {
		t.Fatal("expected credentials on backend config")
	}
}
