package kit

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// EdgeServer is the main component of the kit. It glues all other components of the
// app to each other.
type EdgeServer struct {
	sb        *southBridge
	nb        []*northBridge
	gh        []HandlerFunc
	svc       []Service
	cd        ConnDelegate
	contracts map[string]Contract
	eh        ErrHandlerFunc
	l         Logger
	wg        sync.WaitGroup

	// trace tools
	t Tracer

	// configs
	prefork         bool
	reusePort       bool
	shutdownTimeout time.Duration

	// local store
	ls localStore
}

func NewServer(opts ...Option) *EdgeServer {
	s := &EdgeServer{
		contracts: map[string]Contract{},
		ls: localStore{
			kv: map[string]any{},
		},
	}
	cfg := &edgeConfig{
		logger:     NOPLogger{},
		errHandler: func(ctx *Context, err error) {},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	s.l = cfg.logger
	s.prefork = cfg.prefork
	s.reusePort = cfg.reusePort
	s.eh = cfg.errHandler
	s.gh = cfg.globalHandlers
	s.cd = cfg.connDelegate
	if cfg.tracer != nil {
		s.t = cfg.tracer
	}

	if cfg.cluster != nil {
		s.registerCluster(utils.RandomID(32), cfg.cluster)
	}
	for _, gw := range cfg.gateways {
		s.registerGateway(gw)
	}
	for _, svc := range cfg.services {
		s.registerService(svc)
	}

	return s
}

// RegisterGateway registers a Gateway to our server.
func (s *EdgeServer) registerGateway(gw Gateway) *EdgeServer {
	var th HandlerFunc

	// if tracer is set we inject it to our context pool as the first handler
	if s.t != nil {
		th = s.t.Handler()
	}

	nb := &northBridge{
		ctxPool: ctxPool{
			ls: &s.ls,
			th: th,
		},
		cd: s.cd,
		wg: &s.wg,
		eh: s.eh,
		c:  s.contracts,
		gw: gw,
		sb: s.sb,
	}
	s.nb = append(s.nb, nb)

	// Subscribe the northBridge, which is a GatewayDelegate, to connect northBridge with the Gateway
	gw.Subscribe(nb)

	return s
}

// RegisterCluster registers a Cluster to our server.
func (s *EdgeServer) registerCluster(id string, cb Cluster) *EdgeServer {
	var th HandlerFunc
	if s.t != nil {
		th = s.t.Handler()
	}

	s.sb = &southBridge{
		ctxPool: ctxPool{
			ls: &s.ls,
			th: th,
		},
		id:            id,
		wg:            &s.wg,
		eh:            s.eh,
		c:             s.contracts,
		cb:            cb,
		tp:            s.t,
		inProgressMtx: utils.SpinLock{},
		inProgress:    map[string]chan *envelopeCarrier{},
		msgFactories:  map[string]MessageFactoryFunc{},
		l:             s.l,
	}

	// Subscribe the southBridge, which is a ClusterDelegate, to connect southBridge with the Cluster
	cb.Subscribe(id, s.sb)

	return s
}

// RegisterService registers a Service to our server. We need to define the appropriate
// RouteSelector in each desc.Contract.
func (s *EdgeServer) registerService(svc Service) *EdgeServer {
	if _, ok := s.contracts[svc.Name()]; ok {
		panic(errors.New("service already registered: %s", svc.Name()))
	}

	s.svc = append(s.svc, svc)
	for _, c := range svc.Contracts() {
		s.contracts[c.ID()] = WrapContract(
			c,
			ContractWrapperFunc(s.wrapWithGlobalHandlers),
			ContractWrapperFunc(s.sb.wrapWithCoordinator),
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

	s.l.Debugf("server started.")

	if s.prefork {
		if childID() > 0 {
			s.startChild(ctx)
		} else {
			s.startParent(ctx)
		}

		return s
	}

	s.startup(ctx)

	return s
}

func (s *EdgeServer) startChild(ctx context.Context) {
	s.l.Debugf("child process [%d] with parent [%d] started. ", os.Getpid(), os.Getppid())

	// we are in child process
	// use 1 cpu core per child process
	runtime.GOMAXPROCS(1)

	// kill current child proc when master exits
	go s.watchParent()

	s.startup(ctx)
}

// watchParent watches the parent process
func (s *EdgeServer) watchParent() {
	if runtime.GOOS == "windows" {
		// finds parent process,
		// and waits for it to exit
		p, err := os.FindProcess(os.Getppid())
		if err == nil {
			_, _ = p.Wait()
		}

		s.shutdown(context.Background())

		return
	}
	// if it is equal to 1 (init process ID),
	// it indicates that the master process has exited
	for range time.NewTicker(time.Millisecond * 500).C {
		if os.Getppid() == 1 {
			s.shutdown(context.Background())

			return
		}
	}
}

func (s *EdgeServer) startParent(_ context.Context) {
	// create variables
	maxProc := runtime.GOMAXPROCS(0)

	children := make(map[int]*exec.Cmd)
	childChan := make(chan child, maxProc)

	// launch child processes
	for i := 0; i < maxProc; i++ {
		/* #nosec G204 */
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// add child flag into child proc env
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("%s=%d", envForkChildKey, i+1),
		)
		if err := cmd.Start(); err != nil {
			panic(fmt.Errorf("failed to start a child prefork process, error: %w", err))
		}

		// store child process
		pid := cmd.Process.Pid
		children[pid] = cmd

		// notify master if child crashes
		go func() {
			childChan <- child{pid, cmd.Wait()}
		}()
	}

	ch := <-childChan
	s.l.Debugf("detect child's exit. pid=%d, err=%v", ch.pid, ch.err)

	// if any child exited then we terminate all children and exit program.
	for _, proc := range children {
		_ = proc.Process.Kill()
	}

	os.Exit(0)
}

func (s *EdgeServer) startup(ctx context.Context) {
	for idx := range s.nb {
		for _, svc := range s.svc {
			for _, c := range svc.Contracts() {
				s.nb[idx].gw.Register(svc.Name(), c.ID(), c.Encoding(), c.RouteSelector(), c.Input())
			}
		}

		err := s.nb[idx].gw.Start(
			ctx,
			GatewayStartConfig{
				ReusePort: s.prefork || s.reusePort,
			},
		)
		if err != nil {
			s.l.Errorf("got error on starting gateway: %v", err)
			panic(err)
		}
	}

	if s.sb != nil {
		for _, svc := range s.svc {
			for _, c := range svc.Contracts() {
				s.sb.registerContract(c.Input(), c.Output())
			}
		}
		err := s.sb.Start(ctx)
		if err != nil {
			s.l.Errorf("got error on starting cluster: %v", err)
			panic(err)
		}
	}
}

// Shutdown stops the server. If there is no signal input, then it shut down the server immediately.
// However, if there is one or more signals added in the input argument, then it waits for any of them to
// trigger the shutdown process.
// Since this is a graceful shutdown, it waits for all flying requests to complete. However, you can set
// the maximum time that it waits before forcefully shutting down the server, by WithShutdownTimeout
// option. The Default value is 1 minute.
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

	if s.prefork && childID() == 0 {
		return
	}

	s.shutdown(ctx)
}

func (s *EdgeServer) shutdown(ctx context.Context) {
	// Shutdown all the registered gateways
	for idx := range s.nb {
		err := s.nb[idx].gw.Shutdown(ctx)
		if err != nil {
			s.l.Errorf("got error on shutdown gateway: %v", err)
		}
	}

	if s.sb != nil {
		err := s.sb.Shutdown(ctx)
		if err != nil {
			s.l.Errorf("got error on shutdown cluster: %v", err)
		}
	}

	if s.shutdownTimeout == 0 {
		s.shutdownTimeout = time.Minute
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
	if s.prefork && childID() > 1 {
		return s
	}

	tw := table.NewWriter()
	tw.SuppressEmptyColumns()
	tw.SetStyle(table.StyleRounded)
	style := tw.Style()
	style.Title = table.TitleOptions{
		Align:  text.AlignLeft,
		Colors: text.Colors{text.FgBlack, text.BgWhite},
		Format: text.FormatTitle,
	}
	style.Options.SeparateRows = true
	style.Color.Header = text.Colors{text.Bold, text.FgWhite}

	tw.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:    1,
			AutoMerge: true,
			VAlign:    text.VAlignTop,
			Align:     text.AlignLeft,
			WidthMax:  24,
		},
		{
			Number:    2,
			AutoMerge: true,
			VAlign:    text.VAlignTop,
			Align:     text.AlignLeft,
			WidthMax:  36,
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
			AutoMerge:        true,
			VAlign:           text.VAlignTop,
			Align:            text.AlignLeft,
			WidthMax:         84,
			WidthMaxEnforcer: text.WrapText,
		},
	})

	tw.AppendHeader(
		table.Row{
			"ContractID",
			"Bundle",
			"API",
			"Route | Predicate",
			"Handlers",
		},
	)

	for _, svc := range s.svc {
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
	}
	_, _ = w.Write(utils.S2B(tw.Render()))
	_, _ = w.Write(utils.S2B("\n"))

	if x, ok := w.(interface{ Sync() error }); ok {
		_ = x.Sync()
	} else if x, ok := w.(interface{ Flush() error }); ok {
		_ = x.Flush()
	}

	return s
}

func (s *EdgeServer) PrintRoutesCompact(w io.Writer) *EdgeServer {
	if s.prefork && childID() > 1 {
		return s
	}

	tw := table.NewWriter()
	tw.SuppressEmptyColumns()
	tw.SetStyle(table.StyleRounded)
	style := tw.Style()
	style.Title = table.TitleOptions{
		Align:  text.AlignLeft,
		Colors: text.Colors{text.FgBlack, text.BgWhite},
		Format: text.FormatTitle,
	}
	style.Color.Header = text.Colors{text.Bold, text.FgWhite}

	tw.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:    1,
			AutoMerge: true,
			VAlign:    text.VAlignTop,
			Align:     text.AlignLeft,
			WidthMax:  32,
		},
		{
			Number:   2,
			Align:    text.AlignLeft,
			WidthMax: 12,
		},
		{
			Number:           3,
			Align:            text.AlignLeft,
			WidthMax:         120,
			WidthMaxEnforcer: text.WrapSoft,
		},
	})

	tw.AppendHeader(
		table.Row{
			"Service",
			"API",
			"Route | Predicate",
		},
	)

	for _, svc := range s.svc {
		for _, c := range svc.Contracts() {
			if route := rpcRoute(c.RouteSelector()); route != "" {
				tw.AppendRow(
					table.Row{
						svc.Name(),
						text.FgBlue.Sprint("RPC"),
						route,
					},
				)
			}
			if route := restRoute(c.RouteSelector()); route != "" {
				tw.AppendRow(
					table.Row{
						svc.Name(),
						text.FgGreen.Sprint("REST"),
						route,
					},
				)
			}
		}

		tw.AppendSeparator()
	}
	_, _ = w.Write(utils.S2B(tw.Render()))
	_, _ = w.Write(utils.S2B("\n"))

	if x, ok := w.(interface{ Sync() error }); ok {
		_ = x.Sync()
	} else if x, ok := w.(interface{ Flush() error }); ok {
		_ = x.Flush()
	}

	return s
}

func getFuncName(f HandlerFunc) string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	parts := strings.Split(name, "/")

	return getColor(parts[len(parts)-1]).Sprint(parts[len(parts)-1])
}

func getColor(s string) text.Color {
	c := text.Color(crc32.ChecksumIEEE(utils.S2B(s)) % 7)
	c += text.FgBlack + 1

	return c
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
			getColor(rest.GetMethod()),
			text.Bold,
		}.Sprint(rest.GetMethod()),
		text.Colors{
			text.BgWhite, text.FgBlack,
		}.Sprint(rest.GetPath()),
	)
}

type localStore struct {
	kvl sync.RWMutex
	kv  map[string]any
}

var _ Store = (*localStore)(nil)

func (ls *localStore) Get(key string) any {
	ls.kvl.RLock()
	v := ls.kv[key]
	ls.kvl.RUnlock()

	return v
}

func (ls *localStore) Set(key string, value any) {
	ls.kvl.Lock()
	ls.kv[key] = value
	ls.kvl.Unlock()
}

func (ls *localStore) Delete(key string) {
	ls.kvl.Lock()
	delete(ls.kv, key)
	ls.kvl.Unlock()
}

func (ls *localStore) Exists(key string) bool {
	ls.kvl.RLock()
	_, v := ls.kv[key]
	ls.kvl.RUnlock()

	return v
}

func (ls *localStore) Scan(prefix string, cb func(key string) bool) {
	ls.kvl.RLock()
	defer ls.kvl.RUnlock()

	for k := range ls.kv {
		if strings.HasPrefix(k, prefix) {
			if cb(k) {
				return
			}
		}
	}
}
