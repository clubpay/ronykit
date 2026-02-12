package rony

import (
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/x/apidoc"
)

type Server struct {
	cfg  serverConfig
	edge *kit.EdgeServer
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
	if s.cfg.serveDocsPath != "" {
		var (
			docFS fs.FS
			err   error
		)

		switch s.cfg.serveDocsUI {
		default:
			fallthrough
		case redocUI:
			docFS, err = apidoc.New(s.cfg.serverName, s.cfg.version, "").
				ReDocUI(s.cfg.allServiceDesc()...)
		case swaggerUI:
			docFS, err = apidoc.New(s.cfg.serverName, s.cfg.version, "").
				SwaggerUI(s.cfg.allServiceDesc()...)
		case scalarUI:
			docFS, err = apidoc.New(s.cfg.serverName, s.cfg.version, "").
				ScalarUI(s.cfg.allServiceDesc()...)
		}

		if err != nil {
			return err
		}

		s.cfg.gatewayOpts = append(s.cfg.gatewayOpts, fasthttp.WithServeFS(s.cfg.serveDocsPath, "", docFS))
	}

	opts := []kit.Option{
		kit.WithGateway(s.cfg.Gateways()...),
		kit.WithServiceBuilder(s.cfg.allServiceBuilders()...),
	}

	opts = append(opts, s.cfg.edgeOpts...)

	s.edge = kit.NewServer(opts...)

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	err := s.initEdge()
	if err != nil {
		return err
	}

	s.edge.Start(ctx)

	return nil
}

func (s *Server) Stop(ctx context.Context, signals ...os.Signal) {
	s.edge.Shutdown(ctx, signals...)
}

func (s *Server) PrintRoutes(w io.Writer) {
	s.edge.PrintRoutes(w)
}

func (s *Server) PrintRoutesCompact(w io.Writer) {
	s.edge.PrintRoutesCompact(w)
}

// Run the service in blocking mode. If you need more control over the
// lifecycle of the service, you can use the Start and Stop methods.
func (s *Server) Run(ctx context.Context, signals ...os.Signal) error {
	err := s.Start(ctx)
	if err != nil {
		return err
	}

	s.edge.PrintRoutesCompact(os.Stdout)
	s.Stop(ctx, signals...)

	return nil
}

func (s *Server) GenDoc(_ context.Context, w io.Writer) error {
	return apidoc.New(s.cfg.serverName, s.cfg.version, "").
		WriteSwagTo(w, s.cfg.allServiceDesc()...)
}

func (s *Server) GenDocFile(_ context.Context, filename string) error {
	return apidoc.New(s.cfg.serverName, s.cfg.version, "").
		WriteSwagToFile(filename, s.cfg.allServiceDesc()...)
}

// ExportDesc returns all services descriptions.
func (s *Server) ExportDesc() []desc.ServiceDesc {
	return utils.Map(
		func(src *desc.Service) desc.ServiceDesc {
			return desc.ServiceDescFunc(func() *desc.Service {
				return src
			})
		}, utils.MapToArray(s.cfg.services),
	)
}
