package stub

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path"
	"strings"
)

type Stub struct {
	hostPort  string
	secure    bool
	verifyTLS bool

	httpC http.Client
}

func NewStub() Stub {
	return Stub{
		hostPort:  "",
		secure:    false,
		verifyTLS: false,
	}
}

func (s *Stub) httpBaseURL() string {
	baseURL := strings.Builder{}
	baseURL.WriteString("http")
	if s.secure {
		baseURL.WriteRune('s')
	}
	baseURL.WriteString("://")
	baseURL.WriteString(s.hostPort)

	return baseURL.String()
}

func (s *Stub) HTTP() *httpRequest {
	hc := &httpRequest{
		s:      s,
		reqHdr: map[string]string{},
	}

	return hc
}

type httpRequest struct {
	s            *Stub
	method, path string
	reqBody      io.Reader
	reqHdr       map[string]string
	res          *http.Response
}

func (hc *httpRequest) SetMethod(method string) *httpRequest {
	hc.method = method

	return hc
}

func (hc *httpRequest) SetPath(path string) *httpRequest {
	hc.path = path

	return hc
}

func (hc *httpRequest) SetBody(body []byte) *httpRequest {
	hc.reqBody = bytes.NewBuffer(body)

	return hc
}

func (hc *httpRequest) Do(ctx context.Context) (*httpResponse, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		hc.method, path.Join(hc.s.httpBaseURL(), hc.path),
		hc.reqBody,
	)
	if err != nil {
		return nil, err
	}

	for k, v := range hc.reqHdr {
		httpReq.Header.Set(k, v)
	}

	httpRes, err := hc.s.httpC.Do(httpReq)
	if err != nil {
		return nil, err
	}
	hc.res = httpRes

	res := &httpResponse{
		statusCode: httpRes.StatusCode,
		res:        httpRes,
	}

	return res, nil
}

type httpResponse struct {
	statusCode int
	res        *http.Response
}

func (hr *httpResponse) StatusCode() int { return hr.statusCode }

func (hr *httpResponse) GetHeader(key string) string {
	return hr.res.Header.Get(key)
}

func (hr *httpResponse) GetBody() io.Reader {
	return hr.res.Body
}

func (hr *httpResponse) Close() error {
	return hr.res.Body.Close()
}
