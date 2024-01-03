package silverhttp

import (
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/go-www/silverlining"
	"github.com/go-www/silverlining/h1"
)

const (
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderOrigin                        = "Origin"
	HeaderTimingAllowOrigin             = "Timing-Allow-Origin"
	HeaderXPermittedCrossDomainPolicies = "X-Permitted-Cross-Domain-Policies"
)

type CORSConfig struct {
	AllowedHeaders    []string
	AllowedMethods    []string
	AllowedOrigins    []string
	ExposedHeaders    []string
	IgnoreEmptyOrigin bool
}

type cors struct {
	headers           string
	methods           string
	origins           []string
	exposedHeaders    string
	ignoreEmptyOrigin bool
}

func newCORS(cfg CORSConfig) *cors {
	c := &cors{
		ignoreEmptyOrigin: cfg.IgnoreEmptyOrigin,
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
			MethodGet, MethodHead, MethodPost, MethodPut,
			MethodPatch, MethodConnect, MethodDelete,
			MethodTrace, MethodOptions,
		}, ", ")
	} else {
		c.methods = strings.Join(cfg.AllowedMethods, ", ")
	}

	return c
}

func (cors *cors) handle(rc *httpConn) {
	if cors == nil {
		return
	}

	resHdr := rc.ctx.ResponseHeaders()
	// ByPass cors (Cross Origin Resource Sharing) check
	resHdr.Set("Vary", HeaderOrigin)
	resHdr.Set(HeaderAccessControlExposeHeaders, cors.exposedHeaders)

	origin := rc.Get(HeaderOrigin)
	if cors.origins[0] == "*" {
		resHdr.Set(HeaderAccessControlAllowOrigin, origin)
	} else {
		for _, allowedOrigin := range cors.origins {
			if strings.EqualFold(origin, allowedOrigin) {
				resHdr.Set(HeaderAccessControlAllowOrigin, origin)
			}
		}
	}

	if rc.ctx.Method() == h1.MethodOPTIONS {
		resHdr.Set(
			"Vary",
			strings.Join(
				[]string{
					HeaderAccessControlRequestMethod,
					HeaderAccessControlRequestHeaders,
				},
				",",
			),
		)

		resHdr.Set(HeaderAccessControlRequestMethod, cors.methods)
		reqHeaders, _ := rc.ctx.RequestHeaders().GetBytes(utils.S2B(HeaderAccessControlRequestHeaders))
		if len(reqHeaders) > 0 {
			resHdr.Set(HeaderAccessControlAllowHeaders, utils.B2S(reqHeaders))
		} else {
			resHdr.Set(HeaderAccessControlAllowHeaders, cors.headers)
		}

		resHdr.Set(HeaderAccessControlAllowMethods, cors.methods)
		rc.ctx.WriteHeader(StatusNoContent)
	}
}

func (cors *cors) handleWS(ctx *silverlining.Context) bool {
	if cors == nil {
		return true
	}

	origin, _ := ctx.RequestHeaders().Get(HeaderOrigin)
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
