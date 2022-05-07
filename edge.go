package ronykit

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"

	"github.com/clubpay/ronykit/internal/errors"
	"github.com/clubpay/ronykit/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

var errServiceAlreadyRegistered errors.ErrFunc = func(v ...interface{}) error {
	return fmt.Errorf("service %s already registered", v...)
}

// EdgeServer is the main component of the ronykit. It glues all other components of the
// app to each other.
type EdgeServer struct {
	nb  []*northBridge
	sb  []*southBridge
	svc map[string]Service
	eh  ErrHandler
	l   Logger
}

func NewServer(opts ...Option) *EdgeServer {
	s := &EdgeServer{
		l:   nopLogger{},
		svc: map[string]Service{},
		eh:  func(ctx *Context, err error) {},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RegisterBundle registers a Bundle to our server.
// Currently, two types of Bundles are supported: Gateway and Cluster
func (s *EdgeServer) RegisterBundle(b Bundle) *EdgeServer {
	gw, ok := b.(Gateway)
	if ok {
		nb := &northBridge{
			l:  s.l,
			b:  gw,
			eh: s.eh,
		}
		s.nb = append(s.nb, nb)

		// Subscribe the northBridge, which is a GatewayDelegate, to connect northBridge with the Gateway
		gw.Subscribe(nb)
	}

	c, ok := b.(Cluster)
	if ok {
		sb := &southBridge{
			l:  s.l,
			c:  c,
			eh: s.eh,
		}
		s.sb = append(s.sb, sb)

		// Subscribe the southBridge, which is a ClusterDelegate, to connect southBridge with the Cluster
		c.Subscribe(sb)
	}

	return s
}

// RegisterService registers a Service to our server. We need to define the appropriate
// RouteSelector in each desc.Contract.
func (s *EdgeServer) RegisterService(svc Service) *EdgeServer {
	if _, ok := s.svc[svc.Name()]; ok {
		panic(errServiceAlreadyRegistered(svc.Name()))
	}

	s.svc[svc.Name()] = WrapServiceContracts(svc, ContractWrapperFunc(wrapWithForwarder))

	return s
}

// Start registers services in the registered bundles and start the bundles.
func (s *EdgeServer) Start(ctx context.Context) *EdgeServer {
	if ctx == nil {
		ctx = context.Background()
	}

	for idx := range s.nb {
		for _, svc := range s.svc {
			s.nb[idx].b.Register(svc)
		}

		err := s.nb[idx].b.Start(ctx)
		if err != nil {
			s.l.Errorf("got error on starting gateway: %v", err)
			panic(err)
		}
	}

	s.l.Debug("server started.")

	return s
}

func (s *EdgeServer) Shutdown(ctx context.Context, signals ...os.Signal) {
	if ctx == nil {
		ctx = context.Background()
	}

	if len(signals) > 0 {
		// Create a signal channel and bind it to all the os signals in the arg
		shutdownChan := make(chan os.Signal, 1)
		signal.Notify(shutdownChan, signals...)

		// Wait for the shutdown signal
		<-shutdownChan
	}

	// Start all the registered gateways
	for idx := range s.nb {
		err := s.nb[idx].b.Shutdown(ctx)
		if err != nil {
			s.l.Errorf("got error on shutdown: %v", err)
		}
	}

	return
}

func (s *EdgeServer) PrintRoutes(w io.Writer) *EdgeServer {
	tw := table.NewWriter()
	tw.SuppressEmptyColumns()
	tw.AppendHeader(
		table.Row{
			text.Bold.Sprint("Service"),
			text.Bold.Sprint("Proto"),
			text.Bold.Sprint("Route"),
			text.Bold.Sprint("Handlers"),
		},
		table.RowConfig{AutoMerge: true},
	)

	for _, svc := range s.svc {
		tw.AppendRow(
			table.Row{
				text.Bold.Sprint(svc.Name()),
				"", "", "",
			},
			table.RowConfig{AutoMerge: true},
		)
		colIdx := 0
		for _, c := range svc.Contracts() {
			sb := strings.Builder{}
			for idx, h := range c.Handlers() {
				if idx != 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(getFuncName(h))
			}
			if rs, ok := c.RouteSelector().(RPCRouteSelector); ok {
				if rs.GetPredicate() != "" {
					colIdx++
					tw.AppendRow(
						table.Row{
							colIdx,
							text.FgBlue.Sprint("RPC"),
							text.Colors{
								text.Bold, text.FgBlue,
							}.Sprint(rs.GetPredicate()),
							sb.String(),
						},
						table.RowConfig{AutoMerge: true},
					)
				}
			}

			if rs, ok := c.RouteSelector().(RESTRouteSelector); ok {
				if rs.GetMethod() != "" && rs.GetPath() != "" {
					colIdx++
					tw.AppendRow(
						table.Row{
							colIdx,
							text.FgGreen.Sprint("REST"),
							fmt.Sprintf("%s %s",
								text.Colors{
									text.Bold, text.FgGreen,
								}.Sprint(rs.GetMethod()),
								text.Colors{
									text.BgWhite, text.FgHiWhite,
								}.Sprint(rs.GetPath()),
							),
							sb.String(),
						},
						table.RowConfig{AutoMerge: true},
					)
				}
			}
		}
		tw.AppendSeparator()
	}

	_, _ = w.Write(utils.S2B(tw.Render()))
	_, _ = w.Write(utils.S2B("\n"))

	return s
}

func getFuncName(f HandlerFunc) string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	parts := strings.Split(name, "/")

	c := text.Color(crc32.ChecksumIEEE(utils.S2B(parts[len(parts)-1])) % 7)
	c += text.FgBlack + 1

	return c.Sprint(parts[len(parts)-1])
}
