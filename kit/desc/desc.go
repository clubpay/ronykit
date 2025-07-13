package desc

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
)

type ServiceDesc interface {
	Desc() *Service
}

// ServiceDescFunc is helper utility to convert function to a ServiceDesc interface
type ServiceDescFunc func() *Service

func (f ServiceDescFunc) Desc() *Service {
	return f()
}

type Error struct {
	Code    int
	Item    string
	Message kit.Message
	Meta    MessageMeta
}

func BuildService(desc ServiceDesc) kit.Service {
	return desc.Desc().Build()
}

func ToDesc(svc ...*Service) []ServiceDesc {
	return utils.Map(
		func(src *Service) ServiceDesc {
			return ServiceDescFunc(func() *Service { return src })
		}, svc)
}
