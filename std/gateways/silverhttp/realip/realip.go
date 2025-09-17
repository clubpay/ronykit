package realip

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/go-www/silverlining"
)

// Should use canonical format of the header key s
// https://golang.org/pkg/net/http/#CanonicalHeaderKey
// Header may return multiple IP addresses in the format:
// "client IP, proxy 1 IP, proxy 2 IP", so we take the first one.
var (
	xOriginalForwardedForHeader = http.CanonicalHeaderKey("X-Original-Forwarded-For")
	xForwardedForHeader         = http.CanonicalHeaderKey("X-Forwarded-For")
	xForwardedHeader            = http.CanonicalHeaderKey("X-Forwarded")
	forwardedForHeader          = http.CanonicalHeaderKey("Forwarded-For")
	forwardedHeader             = http.CanonicalHeaderKey("Forwarded")
)

// Standard headers used by Amazon EC2, Heroku, and others
var xClientIPHeader = http.CanonicalHeaderKey("X-Client-IP")

// Nginx proxy/FastCGI
var xRealIPHeader = http.CanonicalHeaderKey("X-Real-IP")

// Cloudflare.
// @see https://support.cloudflare.com/hc/en-us/articles/200170986-How-does-Cloudflare-handle-HTTP-Request-headers-
// CF-Connecting-IP - applied to every request to the origin.
var cfConnectingIPHeader = http.CanonicalHeaderKey("CF-Connecting-IP")

// Fastly CDN and Firebase hosting header when forwared to a cloud function
var fastlyClientIPHeader = http.CanonicalHeaderKey("Fastly-Client-Ip")

// Akamai and Cloudflare
var trueClientIPHeader = http.CanonicalHeaderKey("True-Client-Ip")

var cidrs []*net.IPNet

func init() {
	maxCidrBlocks := []string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	}

	cidrs = make([]*net.IPNet, len(maxCidrBlocks))
	for i, maxCidrBlock := range maxCidrBlocks {
		_, cidr, _ := net.ParseCIDR(maxCidrBlock)
		cidrs[i] = cidr
	}
}

// IsPrivateAddress works by checking if the address is under private CIDR blocks.
// List of private CIDR blocks can be seen on :
//
// https://en.wikipedia.org/wiki/Private_network
//
// https://en.wikipedia.org/wiki/Link-local_address
func IsPrivateAddress(address string) (bool, error) {
	ipAddress := net.ParseIP(address)
	if ipAddress == nil {
		return false, errors.New("address is not valid")
	}

	for i := range cidrs {
		if cidrs[i].Contains(ipAddress) {
			return true, nil
		}
	}

	return false, nil
}

// FromRequest returns client's real public IP address from http request headers.
func FromRequest(ctx *silverlining.Context) string {
	hdr := ctx.RequestHeaders()

	xClientIP, _ := hdr.GetBytes(utils.S2B(xClientIPHeader))
	if xClientIP != nil {
		return string(xClientIP)
	}

	xOriginalForwardedFor, _ := hdr.GetBytes(utils.S2B(xOriginalForwardedForHeader))
	if xOriginalForwardedFor != nil {
		requestIP, err := retrieveForwardedIP(string(xOriginalForwardedFor))
		if err == nil {
			return requestIP
		}
	}

	xForwardedFor, _ := hdr.GetBytes(utils.S2B(xForwardedForHeader))
	if xForwardedFor != nil {
		requestIP, err := retrieveForwardedIP(string(xForwardedFor))
		if err == nil {
			return requestIP
		}
	}

	ip, err := fromSpecialHeaders(ctx)
	if err == nil {
		return ip
	}

	ip, err = fromForwardedHeaders(ctx)
	if err == nil {
		return ip
	}

	var remoteIP string

	remoteAddr := ctx.RemoteAddr().String()

	if strings.ContainsRune(remoteAddr, ':') {
		remoteIP, _, _ = net.SplitHostPort(remoteAddr)
	} else {
		remoteIP = remoteAddr
	}

	return remoteIP
}

func fromSpecialHeaders(ctx *silverlining.Context) (string, error) {
	ipHeaders := [...]string{cfConnectingIPHeader, fastlyClientIPHeader, trueClientIPHeader, xRealIPHeader}
	for _, iplHeader := range ipHeaders {
		if clientIP, _ := ctx.RequestHeaders().GetBytes(utils.S2B(iplHeader)); clientIP != nil {
			return string(clientIP), nil
		}
	}

	return "", errors.New("can't get ip from special headers")
}

func fromForwardedHeaders(ctx *silverlining.Context) (string, error) {
	forwardedHeaders := [...]string{xForwardedHeader, forwardedForHeader, forwardedHeader}
	for _, forwardedHeader := range forwardedHeaders {
		if forwarded, _ := ctx.RequestHeaders().GetBytes(utils.S2B(forwardedHeader)); forwarded != nil {
			clientIP, err := retrieveForwardedIP(string(forwarded))
			if err == nil {
				return clientIP, nil
			}
		}
	}

	return "", errors.New("can't get ip from forwarded headers")
}

func retrieveForwardedIP(forwardedHeader string) (string, error) {
	for _, address := range strings.Split(forwardedHeader, ",") {
		if len(address) > 0 {
			address = strings.TrimSpace(address)
			isPrivate, err := IsPrivateAddress(address)

			switch {
			case !isPrivate && err == nil:
				return address, nil
			case isPrivate && err == nil:
				return "", errors.New("forwarded ip is private")
			default:
				return "", err
			}
		}
	}

	return "", errors.New("empty or invalid forwarded header")
}
