package rony

import (
	"context"
	"os"
	"sync"

	"github.com/clubpay/ronykit/contrib/swagger"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
)

type Server struct {
	cfg  serverConfig
	edge *kit.EdgeServer

	setupOnce sync.Once
}

func NewServer(opts ...ServerOption) *Server {
	cfg := defaultServerConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	srv := &Server{
		cfg: cfg,
	}

	return srv
}

func (s *Server) initEdge() error {
	s.edge = kit.NewServer(
		kit.WithGateway(s.cfg.Gateway()),
		kit.WithServiceDesc(s.cfg.allServices()...),
	)

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.initEdge(); err != nil {
		return err
	}

	s.edge.Start(ctx)

	return nil
}

func (s *Server) Stop(ctx context.Context, signals ...os.Signal) {
	s.edge.Shutdown(ctx, signals...)
}

func (s *Server) Run(ctx context.Context, signals ...os.Signal) error {
	if err := s.Start(ctx); err != nil {
		return err
	}
	s.edge.PrintRoutesCompact(os.Stdout)
	s.Stop(ctx, signals...)

	return nil
}

func (s *Server) SwaggerAPI(filename string) error {
	return swagger.New(s.cfg.name, "v1", "").
		WithTag("json").
		WriteSwagToFile(
			filename,
			utils.Map(
				func(src *desc.Service) desc.ServiceDesc {
					return desc.ServiceDescFunc(func() *desc.Service {
						return src
					})
				}, utils.MapToArray(s.cfg.services),
			)...,
		)
}
