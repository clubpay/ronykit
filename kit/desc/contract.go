package desc

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
)

type Header struct {
	Name     string
	Required bool
}

func OptionalHeader(name string) Header {
	return Header{
		Name:     name,
		Required: false,
	}
}

func RequiredHeader(name string) Header {
	return Header{
		Name:     name,
		Required: true,
	}
}

type RouteSelector struct {
	Name       string
	Selector   kit.RouteSelector
	Deprecated bool
}

func (rs RouteSelector) Deprecate() RouteSelector {
	rs.Deprecated = true

	return rs
}

func Route(name string, selector kit.RouteSelector) RouteSelector {
	return RouteSelector{
		Name:     name,
		Selector: selector,
	}
}

// Contract is the description of the kit.Contract you are going to create.
type Contract struct {
	Name           string
	Encoding       kit.Encoding
	Handlers       []kit.HandlerFunc
	Wrappers       []kit.ContractWrapper
	RouteSelectors []RouteSelector
	EdgeSelector   kit.EdgeSelectorFunc
	Modifiers      []kit.ModifierFunc
	InputHeaders   []Header
	Input          kit.Message
	InputMeta      MessageMeta
	Output         kit.Message
	OutputMeta     MessageMeta
	PossibleErrors []Error
}

func NewContract() *Contract {
	return &Contract{
		Encoding: kit.JSON,
	}
}

// SetName sets the name of the Contract c, it MUST be unique per Service. However, it
// has no operation effect only is helpful with some other tools such as monitoring,
// logging or tracing tools to identity it.
func (c *Contract) SetName(name string) *Contract {
	c.Name = name

	return c
}

// SetEncoding sets the supported encoding for this contract.
func (c *Contract) SetEncoding(enc kit.Encoding) *Contract {
	c.Encoding = enc

	return c
}

// SetInputHeader sets the headers for this Contract. This is an OPTIONAL parameter, which
// could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) SetInputHeader(headers ...Header) *Contract {
	c.InputHeaders = headers

	return c
}

// SetInput sets the accepting message for this Contract. Contracts are bound to one input message.
func (c *Contract) SetInput(m kit.Message, opt ...MessageMetaOption) *Contract {
	meta := MessageMeta{}
	for _, o := range opt {
		o(&meta)
	}
	c.Input = m
	c.InputMeta = meta

	return c
}

// In is an alias for SetInput
func (c *Contract) In(m kit.Message, opt ...MessageMetaOption) *Contract {
	return c.SetInput(m, opt...)
}

// SetOutput sets the outgoing message for this Contract. This is an OPTIONAL parameter, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) SetOutput(m kit.Message, opt ...MessageMetaOption) *Contract {
	meta := MessageMeta{}
	for _, o := range opt {
		o(&meta)
	}
	c.Output = m
	c.OutputMeta = meta

	return c
}

// Out is an alias for SetOutput
func (c *Contract) Out(m kit.Message, opt ...MessageMetaOption) *Contract {
	return c.SetOutput(m, opt...)
}

// AddError sets the possible errors for this Contract. Using this method is OPTIONAL, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) AddError(err kit.ErrorMessage) *Contract {
	c.PossibleErrors = append(
		c.PossibleErrors,
		Error{
			Code:    err.GetCode(),
			Item:    err.GetItem(),
			Message: err,
		},
	)

	return c
}

// AddRoute adds a kit.RouteSelector for this contract.
func (c *Contract) AddRoute(r ...RouteSelector) *Contract {
	c.RouteSelectors = append(c.RouteSelectors, r...)

	return c
}

// AddSelector adds a kit.RouteSelector for this contract. Selectors are bundle-specific.
// Deprecated: use AddRoute instead
func (c *Contract) AddSelector(s ...kit.RouteSelector) *Contract {
	return c.AddRoute(
		utils.Map(
			func(src kit.RouteSelector) RouteSelector {
				return Route("", src)
			}, s,
		)...,
	)
}

// Selector is an alias for AddSelector
// Deprecated: use AddRoute instead
func (c *Contract) Selector(s ...kit.RouteSelector) *Contract {
	return c.AddRoute(
		utils.Map(
			func(src kit.RouteSelector) RouteSelector {
				return Route("", src)
			}, s,
		)...,
	)
}

// AddNamedSelector adds a kit.RouteSelector for this contract and assigns it a unique name.
// In case you need to use auto-generated stub.Stub for your service/contract, this name will
// be used in the generated code.
// Deprecated: use AddRoute instead
func (c *Contract) AddNamedSelector(name string, s kit.RouteSelector) *Contract {
	return c.AddRoute(Route(name, s))
}

// NamedSelector is an alias for AddNamedSelector
// Deprecated: use AddRoute instead
func (c *Contract) NamedSelector(name string, s kit.RouteSelector) *Contract {
	return c.AddRoute(Route(name, s))
}

// SetCoordinator sets a kit.EdgeSelectorFunc for this contract to coordinate requests to
// right kit.EdgeServer instance.
func (c *Contract) SetCoordinator(f kit.EdgeSelectorFunc) *Contract {
	c.EdgeSelector = f

	return c
}

// Coordinator is an alias for SetCoordinator
func (c *Contract) Coordinator(f kit.EdgeSelectorFunc) *Contract {
	return c.SetCoordinator(f)
}

// AddModifier adds a kit.ModifierFunc for this contract. Modifiers are used to modify
// the outgoing kit.Envelope just before sending to the client.
func (c *Contract) AddModifier(m kit.ModifierFunc) *Contract {
	c.Modifiers = append(c.Modifiers, m)

	return c
}

// AddWrapper adds a kit.ContractWrapper for this contract.
func (c *Contract) AddWrapper(wrappers ...kit.ContractWrapper) *Contract {
	c.Wrappers = append(c.Wrappers, wrappers...)

	return c
}

// AddHandler add handler for this contract.
func (c *Contract) AddHandler(h ...kit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers, h...)

	return c
}

// SetHandler set the handler by replacing the already existing ones.
func (c *Contract) SetHandler(h ...kit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers[:0], h...)

	return c
}

// contractImpl is a simple implementation of kit.Contract interface.
type contractImpl struct {
	id             string
	routeSelector  kit.RouteSelector
	memberSelector kit.EdgeSelectorFunc
	handlers       []kit.HandlerFunc
	modifiers      []kit.ModifierFunc
	input          kit.Message
	output         kit.Message
	enc            kit.Encoding
}

var _ kit.Contract = (*contractImpl)(nil)

func (r *contractImpl) setInput(input kit.Message) *contractImpl {
	r.input = input

	return r
}

func (r *contractImpl) setOutput(output kit.Message) *contractImpl {
	r.output = output

	return r
}

func (r *contractImpl) setEncoding(enc kit.Encoding) *contractImpl {
	r.enc = enc

	return r
}

func (r *contractImpl) setRouteSelector(selector kit.RouteSelector) *contractImpl {
	r.routeSelector = selector

	return r
}

func (r *contractImpl) setMemberSelector(selector kit.EdgeSelectorFunc) *contractImpl {
	r.memberSelector = selector

	return r
}

func (r *contractImpl) addHandler(handlers ...kit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers, handlers...)

	return r
}

func (r *contractImpl) setModifier(modifiers ...kit.ModifierFunc) *contractImpl {
	r.modifiers = append(r.modifiers[:0], modifiers...)

	return r
}

func (r *contractImpl) ID() string {
	return r.id
}

func (r *contractImpl) RouteSelector() kit.RouteSelector {
	return r.routeSelector
}

func (r *contractImpl) EdgeSelector() kit.EdgeSelectorFunc {
	return r.memberSelector
}

func (r *contractImpl) Handlers() []kit.HandlerFunc {
	return r.handlers
}

func (r *contractImpl) Modifiers() []kit.ModifierFunc {
	return r.modifiers
}

func (r *contractImpl) Input() kit.Message {
	return r.input
}

func (r *contractImpl) Output() kit.Message {
	return r.output
}

func (r *contractImpl) Encoding() kit.Encoding {
	return r.enc
}
