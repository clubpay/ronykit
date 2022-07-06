package desc

import (
	"github.com/clubpay/ronykit"
)

type ServiceDesc interface {
	Desc() *Service
}

type Error struct {
	Code    int
	Item    string
	Message ronykit.Message
}
