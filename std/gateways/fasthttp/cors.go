package fasthttp

import (
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/valyala/fasthttp"
)

type CORSConfig struct {
	AllowedHeaders    []string
	AllowedMethods    []string
	AllowedOrigins    []string
	ExposedHeaders    []string
	IgnoreEmptyOrigin bool
	AllowCredentials  bool
}

type cors struct {
	headers           string
	methods           string
	origins           []string
	ignoreEmptyOrigin bool
	allowCredentials  bool
	exposedHeaders    string
}

func newCORS(cfg CORSConfig) *cors {
	c := &cors{
		ignoreEmptyOrigin: cfg.IgnoreEmptyOrigin,
		allowCredentials:  cfg.AllowCredentials,
	}
	if len(cfg.AllowedOrigins) == 0 {
		c.origins = []string{"*"}
	} else {
		c.origins = cfg.AllowedOrigins
	}

	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{
			"Origin", "Accept", "Content-Type",
			"X-Requested-With", "X-Auth-Tokens", "Authorization",
		}
	}

	if len(cfg.ExposedHeaders) == 0 {
		c.exposedHeaders = "*"
	} else {
		c.exposedHeaders = strings.Join(cfg.ExposedHeaders, ",")
	}

	c.headers = strings.Join(cfg.AllowedHeaders, ",")
	if len(cfg.AllowedMethods) == 0 {
		c.methods = strings.Join([]string{
			fasthttp.MethodGet, fasthttp.MethodHead, fasthttp.MethodPost, fasthttp.MethodPut,
			fasthttp.MethodPatch, fasthttp.MethodConnect, fasthttp.MethodDelete,
			fasthttp.MethodTrace, fasthttp.MethodOptions,
		}, ", ")
	} else {
		c.methods = strings.Join(cfg.AllowedMethods, ", ")
	}

	return c
}

func (cors *cors) preflightCheck(rc *httpConn) {
	rc.ctx.Request.Header.Peek(fasthttp.HeaderAccessControlRequestMethod)
}

func (cors *cors) handle(ctx *fasthttp.RequestCtx) {
	if cors == nil {
		return
	}

	// ByPass cors (Cross Origin Resource Sharing) check
	ctx.Response.Header.Add("Vary", fasthttp.HeaderOrigin)
	ctx.Response.Header.Set(fasthttp.HeaderAccessControlExposeHeaders, cors.exposedHeaders)

	if cors.allowCredentials {
		ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowCredentials, "true")
	}

	origin := ctx.Request.Header.Peek(fasthttp.HeaderOrigin)
	if cors.origins[0] == "*" {
		ctx.Response.Header.SetBytesV(fasthttp.HeaderAccessControlAllowOrigin, origin)
	} else {
		for _, allowedOrigin := range cors.origins {
			if strings.EqualFold(utils.B2S(origin), allowedOrigin) {
				ctx.Response.Header.SetBytesV(fasthttp.HeaderAccessControlAllowOrigin, origin)
			}
		}
	}

	if ctx.IsOptions() {
		ctx.Response.Header.Add("Vary", fasthttp.HeaderAccessControlRequestMethod)
		ctx.Response.Header.Add("Vary", fasthttp.HeaderAccessControlRequestHeaders)
		ctx.Response.Header.Set(fasthttp.HeaderAccessControlRequestMethod, cors.methods)

		reqHeaders := ctx.Request.Header.Peek(fasthttp.HeaderAccessControlRequestHeaders)
		if len(reqHeaders) > 0 {
			ctx.Response.Header.SetBytesV(fasthttp.HeaderAccessControlAllowHeaders, reqHeaders)
		} else {
			ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowHeaders, cors.headers)
		}

		ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowMethods, cors.methods)
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}
}

func (cors *cors) handleWS(ctx *fasthttp.RequestCtx) bool {
	if cors == nil {
		return true
	}

	origin := utils.B2S(ctx.Request.Header.Peek(fasthttp.HeaderOrigin))
	if origin == "" && cors.ignoreEmptyOrigin {
		return true
	}

	if cors.origins[0] == "*" {
		return true
	} else {
		for _, allowedOrigin := range cors.origins {
			if strings.EqualFold(origin, allowedOrigin) {
				return true
			}
		}
	}

	return false
}
