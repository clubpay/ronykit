package rony

import (
	"context"
	"os"
	"sync"

	"github.com/clubpay/ronykit/kit"
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
