package fasthttp

import (
	"strings"

	"github.com/valyala/fasthttp"
)

type CORSConfig struct {
	AllowedHeaders []string
	AllowedMethods []string
	AllowedOrigins []string
	ExposedHeaders []string
}

type cors struct {
	headers        string
	methods        string
	origins        []string
	exposedHeaders string
}

func newCORS(cfg CORSConfig) *cors {
	c := &cors{}
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

func (cors *cors) handle(rc *httpConn, routeFound bool) {
	if cors == nil {
		return
	}

	// ByPass cors (Cross Origin Resource Sharing) check
	rc.ctx.Response.Header.Add("Vary", fasthttp.HeaderOrigin)
	rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlExposeHeaders, cors.exposedHeaders)

	origin := rc.Get(fasthttp.HeaderOrigin)
	if cors.origins[0] == "*" {
		rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowOrigin, origin)
	} else {
		for _, allowedOrigin := range cors.origins {
			if strings.EqualFold(origin, allowedOrigin) {
				rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowOrigin, origin)
			}
		}
	}

	if routeFound {
		return
	}

	if rc.ctx.IsOptions() {
		rc.ctx.Response.Header.Add("Vary", fasthttp.HeaderAccessControlRequestMethod)
		rc.ctx.Response.Header.Add("Vary", fasthttp.HeaderAccessControlRequestHeaders)
		rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlRequestMethod, cors.methods)
		reqHeaders := rc.ctx.Request.Header.Peek(fasthttp.HeaderAccessControlRequestHeaders)
		if len(reqHeaders) > 0 {
			rc.ctx.Response.Header.SetBytesV(fasthttp.HeaderAccessControlAllowHeaders, reqHeaders)
		} else {
			rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowHeaders, cors.headers)
		}

		rc.ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowMethods, cors.methods)
		rc.ctx.SetStatusCode(fasthttp.StatusNoContent)
	} else {
		rc.ctx.SetStatusCode(fasthttp.StatusNotImplemented)
	}
}
