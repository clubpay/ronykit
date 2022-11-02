package desc

import "github.com/clubpay/ronykit/kit"

func Register(descriptions ...ServiceDesc) kit.Option {
	return func(s *kit.EdgeServer) {
		for _, d := range descriptions {
			s.RegisterService(d.Desc().Generate())
		}
	}
}
