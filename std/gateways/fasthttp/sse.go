package fasthttp

import (
	"bufio"
	"fmt"

	"github.com/valyala/fasthttp"
)

const (
	sseContentType  = "text/event-stream"
	sseEventMessage = "message"
)

func setSSEHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetContentType(sseContentType)
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
}

func writeSSEEvent(w *bufio.Writer, event string, data []byte) error {
	if event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}

	return w.Flush()
}
