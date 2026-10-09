package flow

import (
	"context"
	"crypto/tls"

	"go.temporal.io/sdk/client"
)

// NewAPIKeyStaticCredentials returns credentials that send a fixed API key on
// every request, as a bearer token. TLS is enabled when BackendConfig has not
// already configured it. Rotate the key without reconnecting by using
// NewAPIKeyDynamicCredentials instead.
func NewAPIKeyStaticCredentials(apiKey string) Credentials {
	return client.NewAPIKeyStaticCredentials(apiKey)
}

// NewAPIKeyDynamicCredentials returns credentials that call apiKeyCallback on
// every request. A non-empty key is sent as a bearer token and overrides any
// Authorization metadata already on the context. An empty key leaves that
// metadata unchanged. A non-nil error fails the call. TLS is enabled when
// BackendConfig has not already configured it.
func NewAPIKeyDynamicCredentials(apiKeyCallback func(context.Context) (string, error)) Credentials {
	return client.NewAPIKeyDynamicCredentials(apiKeyCallback)
}

// NewMTLSCredentials returns credentials that authenticate with certificate.
// TLS is enabled when BackendConfig has not already configured it. Client
// creation fails if that TLS config already has a client certificate.
func NewMTLSCredentials(certificate tls.Certificate) Credentials {
	return client.NewMTLSCredentials(certificate)
}
