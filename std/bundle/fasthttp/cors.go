package fasthttp

import (
	"strings"

	"github.com/valyala/fasthttp"
)

const (
	headerOrigin                      = "Origin"
	headerAccessControlAllowOrigin    = "Access-Control-Allow-Origin"
	headerAccessControlAllowHeaders   = "Access-Control-Allow-Headers"
	headerAccessControlAllowMethods   = "Access-Control-Allow-Methods"
	headerAccessControlRequestHeaders = "Access-Control-Request-Headers"
	headerAccessControlRequestMethod  = "Access-Control-Request-Method"
)

type CORSConfig struct {
	AllowedHeaders []string
	AllowedMethods []string
	AllowedOrigins []string
}

type cors struct {
	headers string
	methods string
	origins string
}

func newCORS(cfg CORSConfig) *cors {
	c := &cors{}
	if len(cfg.AllowedOrigins) == 0 {
		c.origins = "*"
	} else {
		c.origins = strings.Join(cfg.AllowedOrigins, ", ")
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{
			"Origin", "Accept", "Content-Type",
			"X-Requested-With", "X-Auth-Tokens", "Authorization",
		}
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
