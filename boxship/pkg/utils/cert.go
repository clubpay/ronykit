package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CertTemplate is a helper function to create a cert template with a serial number and other required fields
func CertTemplate(cn, org, cname string, dnsNames ...string) *x509.Certificate {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(fmt.Errorf("failed to generate serial number: %v", err))
	}

	dnsNames = append(dnsNames, cname)

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{cn},
			Organization: []string{org},
			CommonName:   cname,
		},
		DNSNames:              dnsNames,
		SignatureAlgorithm:    x509.ECDSAWithSHA384,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: false,
	}
}

func RootCATemplate(cn, org, cname string) *x509.Certificate {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(err)
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{cn},
			Organization: []string{org},
			CommonName:   cname,
		},
		NotBefore:             time.Now().Add(-10 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}
}

func CreateCert(
	template, parent *x509.Certificate,
	pub interface{}, parentPriv interface{},
) (cert *x509.Certificate, err error) {
	certDER, err := CreateCertDER(template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)

	return
}

func CreateCertDER(
	template, parent *x509.Certificate,
	pub interface{}, parentPriv interface{},
) ([]byte, error) {
	return x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
}

func CertToPEM(cert *x509.Certificate) []byte {
	// PEM encode the certificate (this is a standard TLS encoding)
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}

	return pem.EncodeToMemory(&b)
}

func GenerateRootCertificate(certTmpl *x509.Certificate, dir string) {
	// generate a new key-pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}

	certDER, err := CreateCertDER(certTmpl, certTmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		panic(err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	certCerPath := filepath.Join(dir, "ca.cer")
	certPEMPath := filepath.Join(dir, "ca.crt")
	certKeyPath := filepath.Join(dir, "ca.key")

	_ = os.WriteFile(certPEMPath, CertToPEM(cert), os.ModePerm)
	_ = os.WriteFile(certCerPath, certDER, os.ModePerm)
	_ = os.WriteFile(certKeyPath, keyPEM, os.ModePerm)
}

func GenerateSelfSignedCert(certTmpl *x509.Certificate, keyPath, certPath string) {
	// generate a new key-pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// describe what the certificate will be used for
	cert, err := CreateCert(certTmpl, certTmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	_ = os.WriteFile(certPath, CertToPEM(cert), os.ModePerm)
	_ = os.WriteFile(keyPath, keyPEM, os.ModePerm)
}

func GenerateCert(certTmpl, caTmpl *x509.Certificate, caKeyPath, keyPath, certPath string) {
	// generate a new key-pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}

	caKeyBytes, err := os.ReadFile(caKeyPath)
	if err != nil {
		panic(err)
	}
	b, _ := pem.Decode(caKeyBytes)
	caPrivateKey, err := x509.ParseECPrivateKey(b.Bytes)
	if err != nil {
		panic(err)
	}

	// describe what the certificate will be used for
	cert, err := CreateCert(certTmpl, caTmpl, &privateKey.PublicKey, caPrivateKey)
	if err != nil {
		panic(err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	// PEM encode the private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	err = os.WriteFile(certPath, CertToPEM(cert), os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(keyPath, keyPEM, os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func GetCertificate(keyPath, certPath string) (*tls.Certificate, error) {
	keyPEM, _ := os.ReadFile(keyPath)
	certPEM, _ := os.ReadFile(certPath)
	// Create a TLS cert using the private key and certificate
	rootTLSCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &rootTLSCert, nil
}
