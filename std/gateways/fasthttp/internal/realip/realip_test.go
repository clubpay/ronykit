package realip_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/clubpay/ronykit/std/gateways/fasthttp/internal/realip"
	"github.com/valyala/fasthttp"
)

func TestIsPrivateAddr(t *testing.T) {
	testData := map[string]bool{
		"127.0.0.0":   true,
		"10.0.0.0":    true,
		"169.254.0.0": true,
		"192.168.0.0": true,
		"::1":         true,
		"fc00::":      true,

		"172.15.0.0": false,
		"172.16.0.0": true,
		"172.31.0.0": true,
		"172.32.0.0": false,

		"147.12.56.11": false,
	}

	for addr, isLocal := range testData {
		isPrivate, err := realip.IsPrivateAddress(addr)
		if err != nil {
			t.Errorf("fail processing %s: %v", addr, err)
		}

		if isPrivate != isLocal {
			format := "%s should "
			if !isLocal {
				format += "not "
			}
			format += "be local address"

			t.Errorf(format, addr)
		}
	}
}

type testIP struct {
	name     string
	request  *fasthttp.RequestCtx
	expected string
}

func TestRealIP(t *testing.T) {
	newRequest := func(remoteAddr string, headers map[string]string) *fasthttp.RequestCtx {
		var ctx fasthttp.RequestCtx

		addr := &net.TCPAddr{
			IP: net.ParseIP(remoteAddr),
		}
		ctx.Init(&ctx.Request, addr, nil)

		for header, value := range headers {
			ctx.Request.Header.Set(header, value)
		}

		return &ctx
	}

	testData := []testIP{
		{
			name:     "No header",
			request:  newRequest("144.12.54.87", map[string]string{}),
			expected: "144.12.54.87",
		},
		{
			name:     "Has X-Forwarded-For",
			request:  newRequest("", map[string]string{"X-Forwarded-For": "144.12.54.87"}),
			expected: "144.12.54.87",
		},
		{
			name: "Has multiple X-Forwarded-For",
			request: newRequest("", map[string]string{
				"X-Forwarded-For": fmt.Sprintf("%s,%s,%s", "119.14.55.11", "144.12.54.87", "127.0.0.0"),
			}),
			expected: "119.14.55.11",
		},
		{
			name:     "Has X-Real-IP",
			request:  newRequest("", map[string]string{"X-Real-IP": "144.12.54.87"}),
			expected: "144.12.54.87",
		},
		{
			name: "Has multiple X-Forwarded-For and X-Real-IP",
			request: newRequest("", map[string]string{
				"X-Real-IP":       "119.14.55.11",
				"X-Forwarded-For": fmt.Sprintf("%s,%s", "144.12.54.87", "127.0.0.0"),
			}),
			expected: "144.12.54.87",
		},
	}

	// Run test
	for _, v := range testData {
		if actual := realip.FromRequest(v.request); v.expected != actual {
			t.Errorf("%s: expected %s but get %s", v.name, v.expected, actual)
		}
	}
}
