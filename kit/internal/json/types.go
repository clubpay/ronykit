package json

import (
	"github.com/goccy/go-json"
)

type RawMessage = json.RawMessage

func Marshal(v any) ([]byte, error) {
	return json.MarshalNoEscape(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v any) error {
	return json.UnmarshalNoEscape(data, v)
}
