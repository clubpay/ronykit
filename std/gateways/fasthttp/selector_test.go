package fasthttp

import (
	"testing"
)

func TestConvertLegacyPath(t *testing.T) {
	in := map[string]string{
		"/echo/:requestID":            "/echo/{requestID}",
		"/echo/:requestID/":           "/echo/{requestID}/",
		"/echo/:requestID/:randomID":  "/echo/{requestID}/{randomID}",
		"/echo/:requestID/:randomID/": "/echo/{requestID}/{randomID}/",
		"/:x/:y/:z":                   "/{x}/{y}/{z}",
		"/something/:x/another/:y":    "/something/{x}/another/{y}",
		"/x/:y/:z":                    "/x/{y}/{z}",
	}
	for k := range in {
		if convertLegacyPathFormat(k) != in[k] {
			t.Fatalf("expected: %s, got: %s", in[k], convertLegacyPathFormat(k))
		}
	}

}
