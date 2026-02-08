package realip

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestRetrieveForwardedIPErrors(t *testing.T) {
	if _, err := retrieveForwardedIP(""); err == nil {
		t.Fatalf("expected empty header error")
	}
	if _, err := retrieveForwardedIP("10.0.0.1"); err == nil {
		t.Fatalf("expected private ip error")
	}
	if _, err := retrieveForwardedIP("not-an-ip"); err == nil {
		t.Fatalf("expected invalid ip error")
	}
}

func TestFromSpecialHeaders(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set(cfConnectingIPHeader, "1.2.3.4")
	got, err := fromSpecialHeaders(ctx)
	if err != nil || got != "1.2.3.4" {
		t.Fatalf("unexpected special header result: %s %v", got, err)
	}
}

func TestFromForwardedHeaders(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set(xForwardedHeader, "192.168.0.1")
	if _, err := fromForwardedHeaders(ctx); err == nil {
		t.Fatalf("expected forwarded header error")
	}
}

func TestIsPrivateAddressInvalid(t *testing.T) {
	if _, err := IsPrivateAddress("bad"); err == nil {
		t.Fatalf("expected invalid ip error")
	}
}

func TestFromRequestXClientIP(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set(xClientIPHeader, "11.22.33.44")
	if got := FromRequest(ctx); got != "11.22.33.44" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}
