package stub

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
)

type testLogger struct{}

func (testLogger) Debugf(_ string, _ ...any) {}
func (testLogger) Errorf(_ string, _ ...any) {}

type testCodec struct{}

func (testCodec) Encode(_ kit.Message, _ io.Writer) error { return nil }
func (testCodec) Marshal(_ any) ([]byte, error)           { return []byte("ok"), nil }
func (testCodec) Decode(_ kit.Message, _ io.Reader) error { return nil }
func (testCodec) Unmarshal(_ []byte, _ any) error         { return nil }

func TestOptionsApplyToConfig(t *testing.T) {
	cfg := config{
		rootCAs: x509.NewCertPool(),
	}

	Secure()(&cfg)
	SkipTLSVerify()(&cfg)
	Name("client")(&cfg)
	WithReadTimeout(time.Second)(&cfg)
	WithWriteTimeout(time.Second * 2)(&cfg)
	WithDialTimeout(time.Second * 3)(&cfg)
	WithMessageCodec(testCodec{})(&cfg)
	WithLogger(testLogger{})(&cfg)
	WithTracePropagator(nil)(&cfg)

	if !cfg.secure || !cfg.skipVerifyTLS || cfg.name != "client" {
		t.Fatalf("unexpected basic config values: %+v", cfg)
	}
	if cfg.readTimeout != time.Second || cfg.writeTimeout != time.Second*2 || cfg.dialTimeout != time.Second*3 {
		t.Fatalf("unexpected timeout config values: %+v", cfg)
	}
	if cfg.codec == nil || cfg.l == nil {
		t.Fatal("expected codec and logger to be set")
	}

	var buf bytes.Buffer
	DumpTo(&buf)(&cfg)
	if cfg.dumpReq != &buf || cfg.dumpRes != &buf {
		t.Fatalf("unexpected dump writers")
	}
}

func TestProxyOptions(t *testing.T) {
	cfg := config{rootCAs: x509.NewCertPool()}

	WithHTTPProxy("localhost:9050", time.Second)(&cfg)
	if cfg.proxy == nil || cfg.dialFunc == nil {
		t.Fatalf("expected http proxy settings")
	}

	WithSocksProxy("localhost:9050")(&cfg)
	if cfg.proxy == nil || cfg.dialFunc == nil {
		t.Fatalf("expected socks proxy settings")
	}
}

func TestCertificateOptions(t *testing.T) {
	cert, pemBytes := generateTestCert(t)

	cfg := config{
		rootCAs: x509.NewCertPool(),
	}

	AddRootCA(cert)(&cfg)
	if len(cfg.rootCAs.Subjects()) == 0 {
		t.Fatal("expected root CAs to include certificate")
	}

	AddRootCAFromPEM(pemBytes)(&cfg)
	if len(cfg.rootCAs.Subjects()) == 0 {
		t.Fatal("expected root CAs from pem")
	}

	newPool := x509.NewCertPool()
	WithCertificatePool(newPool)(&cfg)
	if cfg.rootCAs != newPool {
		t.Fatal("expected certificate pool to be replaced")
	}
}

func generateTestCert(t *testing.T) (*x509.Certificate, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "stub-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:         true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	return cert, pemBytes
}
