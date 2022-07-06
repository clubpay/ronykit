package desc

import "github.com/clubpay/ronykit"

// Contract is the description of the ronykit.Contract you are going to create.
type Contract struct {
	Name           string
	Encoding       ronykit.Encoding
	Handlers       []ronykit.HandlerFunc
	Wrappers       []ronykit.ContractWrapper
	RouteSelectors []ronykit.RouteSelector
	routeNames     []string
	EdgeSelector   ronykit.EdgeSelectorFunc
	Modifiers      []ronykit.Modifier
	Input          ronykit.Message
	Output         ronykit.Message
	PossibleErrors []Error
}

func NewContract() *Contract {
	return &Contract{}
}

// SetName sets the name of the Contract c, it MUST be unique per Service. However, it
// has no operation effect only is helpful with some other tools such as monitoring,
// logging or tracing tools to identity it.
func (c *Contract) SetName(name string) *Contract {
	c.Name = name

	return c
}

// SetEncoding sets the supported encoding for this contract.
func (c *Contract) SetEncoding(enc ronykit.Encoding) *Contract {
	c.Encoding = enc

	return c
}

// SetInput sets the accepting message for this Contract. Contracts are bound to one input message.
// In some odd cases if you need to handle multiple input messages, then you SHOULD create multiple
// contracts but with same handlers and/or selectors.
func (c *Contract) SetInput(m ronykit.Message) *Contract {
	c.Input = m

	return c
}

// SetOutput sets the outgoing message for this Contract. This is an OPTIONAL parameter, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) SetOutput(m ronykit.Message) *Contract {
	c.Output = m

	return c
}

// AddPossibleError sets the possible errors for this Contract. Using this method is OPTIONAL, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
// @Deprecated use AddError instead.
func (c *Contract) AddPossibleError(code int, item string, m ronykit.Message) *Contract {
	c.PossibleErrors = append(c.PossibleErrors, Error{
		Code:    code,
		Item:    item,
		Message: m,
	})

	return c
}

// AddError sets the possible errors for this Contract. Using this method is OPTIONAL, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) AddError(err ronykit.ErrorMessage) *Contract {
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

// AddSelector adds a ronykit.RouteSelector for this contract. Selectors are bundle specific.
func (c *Contract) AddSelector(s ronykit.RouteSelector) *Contract {
	c.RouteSelectors = append(c.RouteSelectors, s)
	c.routeNames = append(c.routeNames, "")

	return c
}

// AddNamedSelector adds a ronykit.RouteSelector for this contract, and assign it a unique name.
// In case of you need to use auto-generated stub.Stub for your service/contract this name will
// be used in the generated code.
func (c *Contract) AddNamedSelector(name string, s ronykit.RouteSelector) *Contract {
	c.RouteSelectors = append(c.RouteSelectors, s)
	c.routeNames = append(c.routeNames, name)

	return c
}

// SetCoordinator sets a ronykit.EdgeSelectorFunc for this contract, to coordinate requests to
// right ronykit.EdgeServer instance.
func (c *Contract) SetCoordinator(f ronykit.EdgeSelectorFunc) *Contract {
	c.EdgeSelector = f

	return c
}

// AddModifier adds a ronykit.Modifier for this contract. Modifiers are used to modify
// the outgoing ronykit.Envelope just before sending to the client.
func (c *Contract) AddModifier(m ronykit.Modifier) *Contract {
	c.Modifiers = append(c.Modifiers, m)

	return c
}

// AddWrapper adds a ronykit.ContractWrapper for this contract.
func (c *Contract) AddWrapper(wrappers ...ronykit.ContractWrapper) *Contract {
	c.Wrappers = append(c.Wrappers, wrappers...)

	return c
}

// AddHandler add handler for this contract.
func (c *Contract) AddHandler(h ...ronykit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers, h...)

	return c
}

// SetHandler set the handler by replacing the already existing ones.
func (c *Contract) SetHandler(h ...ronykit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers[:0], h...)

	return c
}

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	id             string
	routeSelector  ronykit.RouteSelector
	memberSelector ronykit.EdgeSelectorFunc
	handlers       []ronykit.HandlerFunc
	modifiers      []ronykit.Modifier
	input          ronykit.Message
	enc            ronykit.Encoding
}

var _ ronykit.Contract = (*contractImpl)(nil)

func (r *contractImpl) setInput(input ronykit.Message) *contractImpl {
	r.input = input

	return r
}

func (r *contractImpl) setEncoding(enc ronykit.Encoding) *contractImpl {
	r.enc = enc

	return r
}

func (r *contractImpl) setRouteSelector(selector ronykit.RouteSelector) *contractImpl {
	r.routeSelector = selector

	return r
}

func (r *contractImpl) setMemberSelector(selector ronykit.EdgeSelectorFunc) *contractImpl {
	r.memberSelector = selector

	return r
}

func (r *contractImpl) setHandler(handlers ...ronykit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *contractImpl) addHandler(handlers ...ronykit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers, handlers...)

	return r
}

func (r *contractImpl) setModifier(modifiers ...ronykit.Modifier) *contractImpl {
	r.modifiers = append(r.modifiers[:0], modifiers...)

	return r
}

func (r *contractImpl) ID() string {
	return r.id
}

func (r *contractImpl) RouteSelector() ronykit.RouteSelector {
	return r.routeSelector
}

func (r *contractImpl) EdgeSelector() ronykit.EdgeSelectorFunc {
	return r.memberSelector
}

func (r *contractImpl) Handlers() []ronykit.HandlerFunc {
	return r.handlers
}

func (r *contractImpl) Modifiers() []ronykit.Modifier {
	return r.modifiers
}

func (r *contractImpl) Input() ronykit.Message {
	return r.input
}

func (r *contractImpl) Encoding() ronykit.Encoding {
	return r.enc
}
