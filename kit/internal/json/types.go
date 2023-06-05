package json

import (
	"encoding/json"

	"github.com/bytedance/sonic"
)

type RawMessage = json.RawMessage

func Marshal(v any) ([]byte, error) {
	return sonic.ConfigDefault.Marshal(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return sonic.ConfigDefault.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v any) error {
	return sonic.ConfigDefault.Unmarshal(data, v)
}
