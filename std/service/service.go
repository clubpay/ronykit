package service

import "github.com/ronaksoft/ronykit"

// Service is a simple implementation of ronykit.Service interface.
type Service struct {
	name   string
	pre    []ronykit.Handler
	post   []ronykit.Handler
	routes []ronykit.Contract
}

func New(name string) *Service {
	s := &Service{
		name: name,
	}

	return s
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Contracts() []ronykit.Contract {
	return s.routes
}

func (s *Service) PreHandlers() []ronykit.Handler {
	return s.pre
}

func (s *Service) PostHandlers() []ronykit.Handler {
	return s.post
}

func (s *Service) SetPreHandlers(h ...ronykit.Handler) *Service {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *Service) SetPostHandlers(h ...ronykit.Handler) *Service {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *Service) AddContract(contracts ...ronykit.Contract) *Service {
	s.routes = append(s.routes, contracts...)

	return s
}
