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
	"sync"
	"time"

	"github.com/clubpay/ronykit/internal/errors"
	"github.com/clubpay/ronykit/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type contractResolver func(contractID string) Contract

// EdgeServer is the main component of the ronykit. It glues all other components of the
// app to each other.
type EdgeServer struct {
	sb        *southBridge
	nb        []*northBridge
	gh        []HandlerFunc
	svc       []Service
	contracts map[string]Contract
	eh        ErrHandler
	l         Logger
	wg        sync.WaitGroup

	// configs
	shutdownTimeout time.Duration
}

func NewServer(opts ...Option) *EdgeServer {
	s := &EdgeServer{
		l:         nopLogger{},
		contracts: map[string]Contract{},
		eh:        func(ctx *Context, err error) {},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *EdgeServer) getContract(contractID string) Contract {
	return s.contracts[contractID]
}

// RegisterBundle registers a Bundle to our server.
// Currently, two types of Bundles are supported: Gateway and Cluster
func (s *EdgeServer) RegisterBundle(b Bundle) *EdgeServer {
	gw, ok := b.(Gateway)
	if ok {
		nb := &northBridge{
			l:  s.l,
			eh: s.eh,
			cr: s.getContract,
			gw: gw,
			wg: &s.wg,
		}
		s.nb = append(s.nb, nb)

		// Subscribe the northBridge, which is a GatewayDelegate, to connect northBridge with the Gateway
		gw.Subscribe(nb)
	}

	return s
}

func (s *EdgeServer) RegisterClusterBackend(cb ClusterBackend) *EdgeServer {
	s.sb = &southBridge{
		l:          s.l,
		eh:         s.eh,
		cr:         s.getContract,
		wg:         &s.wg,
		cb:         cb,
		inProgress: map[string]chan *envelopeCarrier{},
	}

	return s
}

// RegisterService registers a Service to our server. We need to define the appropriate
// RouteSelector in each desc.Contract.
func (s *EdgeServer) RegisterService(svc Service) *EdgeServer {
	if _, ok := s.contracts[svc.Name()]; ok {
		panic(errors.New("service already registered: %s", svc.Name()))
	}

	s.svc = append(s.svc, svc)
	for _, c := range svc.Contracts() {
		s.contracts[c.ID()] = WrapContract(
			c,
			ContractWrapperFunc(s.wrapWithGlobalHandlers),
			ContractWrapperFunc(wrapWithCoordinator),
		)
	}

	return s
}

func (s *EdgeServer) wrapWithGlobalHandlers(c Contract) Contract {
	if len(s.gh) == 0 {
		return c
	}

	cw := &contractWrap{
		Contract: c,
		h:        s.gh,
	}

	return cw
}

// Start registers services in the registered bundles and start the bundles.
func (s *EdgeServer) Start(ctx context.Context) *EdgeServer {
	if ctx == nil {
		ctx = context.Background()
	}

	for idx := range s.nb {
		for _, svc := range s.svc {
			for _, c := range svc.Contracts() {
				s.nb[idx].gw.Register(svc.Name(), c.ID(), c.Encoding(), c.RouteSelector(), c.Input())
			}
		}

		err := s.nb[idx].gw.Start(ctx)
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
		err := s.nb[idx].gw.Shutdown(ctx)
		if err != nil {
			s.l.Errorf("got error on shutdown: %v", err)
		}
	}

	if s.shutdownTimeout == 0 {
		s.wg.Wait()

		return
	}

	waitCh := make(chan struct{}, 1)
	go func() {
		s.wg.Wait()
		waitCh <- struct{}{}
	}()

	select {
	case <-waitCh:
	case <-time.After(s.shutdownTimeout):
	}
}

func (s *EdgeServer) PrintRoutes(w io.Writer) *EdgeServer {
	for _, svc := range s.svc {
		tw := table.NewWriter()
		tw.SuppressEmptyColumns()
		tw.SetStyle(table.StyleRounded)
		tw.Style().Title.Colors = text.Colors{text.FgBlack, text.BgWhite}
		tw.Style().Format.Header = text.FormatTitle
		tw.Style().Options.SeparateRows = true
		tw.SetColumnConfigs([]table.ColumnConfig{
			{
				Number:   1,
				Align:    text.AlignLeft,
				WidthMax: 24,
			},
			{
				Number:   2,
				Align:    text.AlignLeft,
				WidthMax: 36,
			},
			{
				Number:   3,
				Align:    text.AlignLeft,
				WidthMax: 12,
			},
			{
				Number:           4,
				Align:            text.AlignLeft,
				WidthMax:         52,
				WidthMaxEnforcer: text.WrapSoft,
			},
			{
				Number:           5,
				Align:            text.AlignLeft,
				WidthMax:         84,
				WidthMaxEnforcer: text.WrapSoft,
			},
		})
		tw.AppendHeader(
			table.Row{
				text.Bold.Sprint("ContractID"),
				text.Bold.Sprint("Bundle"),
				text.Bold.Sprint("API"),
				text.Bold.Sprint("Route | Predicate"),
				text.Bold.Sprint("Handlers"),
			},
		)
		tw.SetTitle(text.Bold.Sprint(svc.Name()))

		for _, c := range svc.Contracts() {
			if route := rpcRoute(c.RouteSelector()); route != "" {
				tw.AppendRow(
					table.Row{
						c.ID(),
						reflect.TypeOf(c.RouteSelector()).String(),
						text.FgBlue.Sprint("RPC"),
						route,
						getHandlers(c.Handlers()...),
					},
				)
			}
			if route := restRoute(c.RouteSelector()); route != "" {
				tw.AppendRow(
					table.Row{
						c.ID(),
						reflect.TypeOf(c.RouteSelector()).String(),
						text.FgGreen.Sprint("REST"),
						route,
						getHandlers(c.Handlers()...),
					},
				)
			}
		}

		tw.AppendSeparator()
		_, _ = w.Write(utils.S2B(tw.Render()))
		_, _ = w.Write(utils.S2B("\n"))
	}

	return s
}

func getFuncName(f HandlerFunc) string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	parts := strings.Split(name, "/")

	c := text.Color(crc32.ChecksumIEEE(utils.S2B(parts[len(parts)-1])) % 7)
	c += text.FgBlack + 1

	return c.Sprint(parts[len(parts)-1])
}

func getHandlers(handlers ...HandlerFunc) string {
	sb := strings.Builder{}
	for idx, h := range handlers {
		if idx != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(getFuncName(h))
	}

	return text.WrapSoft(sb.String(), 32)
}

func rpcRoute(rs RouteSelector) string {
	rpc, ok := rs.(RPCRouteSelector)
	if !ok || rpc.GetPredicate() == "" {
		return ""
	}

	return text.Colors{
		text.Bold, text.FgBlue,
	}.Sprint(rpc.GetPredicate())
}

func restRoute(rs RouteSelector) string {
	rest, ok := rs.(RESTRouteSelector)
	if !ok || rest.GetMethod() == "" || rest.GetPath() == "" {
		return ""
	}

	return fmt.Sprintf("%s %s",
		text.Colors{
			text.Bold, text.FgGreen,
		}.Sprint(rest.GetMethod()),
		text.Colors{
			text.BgWhite, text.FgBlack,
		}.Sprint(rest.GetPath()),
	)
}
