package desc

import "github.com/clubpay/ronykit"

func Register(descriptions ...Desc) ronykit.Option {
	return func(s *ronykit.EdgeServer) {
		for _, d := range descriptions {
			s.RegisterService(d.Desc().Generate())
		}
	}
}
