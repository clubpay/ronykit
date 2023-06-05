package json

import (
	"github.com/bytedance/sonic"
)

type Marshaler struct {
	api sonic.API
}

func NewMarshaler() Marshaler {
	return Marshaler{
		api: sonic.ConfigDefault,
	}
}

func (jm Marshaler) Marshal(m any) ([]byte, error) {
	return jm.api.Marshal(m)
}

func (jm Marshaler) Unmarshal(data []byte, m any) error {
	return jm.api.Unmarshal(data, m)
}
