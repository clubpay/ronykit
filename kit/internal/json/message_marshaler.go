package json

import (
	"github.com/goccy/go-json"
)

type Marshaler struct{}

func NewMarshaler() Marshaler {
	return Marshaler{}
}

func (jm Marshaler) Marshal(m any) ([]byte, error) {
	return json.MarshalNoEscape(m)
}

func (jm Marshaler) Unmarshal(data []byte, m any) error {
	return json.Unmarshal(data, m)
}
