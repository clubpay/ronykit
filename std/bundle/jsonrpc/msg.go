package jsonrpc

import "github.com/goccy/go-json"

type Envelope struct {
	Header    map[string]string `json:"hdr"`
	Predicate string            `json:"predicate"`
	ID        int64             `json:"id"`
	Payload   json.RawMessage   `json:"payload"`
}
