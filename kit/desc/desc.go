package desc

import "github.com/clubpay/ronykit/kit"

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
}

func BuildService(desc ServiceDesc) kit.Service {
	return desc.Desc().Build()
}
