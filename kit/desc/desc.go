package desc

import "github.com/clubpay/ronykit/kit"

type ServiceDesc interface {
	Desc() *Service
}

type ServiceDescFunc func() *Service

func (f ServiceDescFunc) Desc() *Service {
	return f()
}

type Error struct {
	Code    int
	Item    string
	Message kit.Message
}

func GenerateService(desc ServiceDesc) kit.Service {
	return desc.Desc().Generate()
}
