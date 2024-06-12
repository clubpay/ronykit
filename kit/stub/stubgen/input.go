package stubgen

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
)

type Input struct {
	tags        []string
	restMethods []RESTMethod
	rpcMethods  []RPCMethod
	messages    map[string]desc.ParsedMessage
	name        string
	pkg         string
}

func NewInput(name, pkg string, descs ...desc.ServiceDesc) *Input {
	in := &Input{
		pkg:  pkg,
		name: name,
	}
	for _, serviceDesc := range descs {
		ps := desc.Parse(serviceDesc)
		in.AddMessage(ps.Messages()...)
		for _, c := range ps.Contracts {
			in.addContract(c)
		}
	}

	return in
}

func (in *Input) AddTags(tags ...string) {
	in.tags = append(in.tags, tags...)
}

func (in *Input) Tags() []string {
	return in.tags
}

func (in *Input) addContract(c desc.ParsedContract) {
	if c.Method != "" && c.Path != "" {
		in.restMethods = append(in.restMethods, RESTMethod{
			Name:      c.Name,
			Method:    c.Method,
			Path:      c.Path,
			Encoding:  c.Encoding,
			Request:   c.Request,
			Responses: c.Responses,
		})
	}
	if c.Predicate != "" {
		in.rpcMethods = append(in.rpcMethods, RPCMethod{
			Name:      c.Name,
			Predicate: c.Predicate,
			Request:   c.Request,
			Responses: c.Responses,
			Encoding:  c.Encoding,
		})
	}
}

func (in *Input) RESTMethods() []RESTMethod {
	return in.restMethods
}

func (in *Input) RPCMethods() []RPCMethod {
	return in.rpcMethods
}

func (in *Input) Messages() map[string]desc.ParsedMessage {
	return in.messages
}

func (in *Input) DTOs() map[string]desc.ParsedMessage {
	return in.messages
}

func (in *Input) Name() string {
	return in.name
}

func (in *Input) Pkg() string {
	return in.pkg
}

func (in *Input) AddMessage(msg ...desc.ParsedMessage) {
	if in.messages == nil {
		in.messages = make(map[string]desc.ParsedMessage)
	}

	for _, m := range msg {
		in.messages[m.Name] = m
	}
}

// RESTMethod represents description of a Contract with kit.RESTRouteSelector.
type RESTMethod struct {
	Name      string
	Method    string
	Path      string
	Encoding  string
	Request   desc.ParsedRequest
	Responses []desc.ParsedResponse
}

func (rm *RESTMethod) GetOKResponse() desc.ParsedResponse {
	return utils.Filter(
		func(src desc.ParsedResponse) bool {
			return !src.IsError()
		}, rm.Responses,
	)[0]
}

func (rm *RESTMethod) GetErrors() []desc.ParsedResponse {
	return utils.Filter(
		func(src desc.ParsedResponse) bool {
			return src.IsError()
		}, rm.Responses,
	)
}

// RPCMethod represents description of a Contract with kit.RPCRouteSelector
type RPCMethod struct {
	Name      string
	Predicate string
	Request   desc.ParsedRequest
	Responses []desc.ParsedResponse
	Encoding  string
	kit.IncomingRPCContainer
	kit.OutgoingRPCContainer
}

func (rm *RPCMethod) GetOKResponse() desc.ParsedResponse {
	return utils.Filter(
		func(src desc.ParsedResponse) bool {
			return !src.IsError()
		}, rm.Responses,
	)[0]
}

func (rm *RPCMethod) GetErrors() []desc.ParsedResponse {
	return utils.Filter(
		func(src desc.ParsedResponse) bool {
			return src.IsError()
		}, rm.Responses,
	)
}
