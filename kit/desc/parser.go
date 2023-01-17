package desc

import (
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit/kit"
)

type ParsedService struct {
	Origin    *Service
	Contracts []ParsedContract
}

type ContractType string

const (
	REST ContractType = "REST"
	RPC  ContractType = "RPC"
)

type ParsedContract struct {
	GroupName string
	Name      string
	Encoding  string

	Type      ContractType
	Path      string
	Method    string
	Predicate string

	Request   ParsedRequest
	Responses []ParsedResponse
}

type ParsedMessage struct {
	Name   string
	Params []ParsedParam
}

type ParsedRequest struct {
	Message ParsedMessage
}

type ParsedResponse struct {
	Message ParsedMessage
	ErrCode int
	ErrItem string
}

func (pr ParsedResponse) IsError() bool {
	return pr.ErrCode != 0
}

type ParamKind string

const (
	None    ParamKind = ""
	String  ParamKind = "string"
	Integer ParamKind = "integer"
	Float   ParamKind = "float"
	Object  ParamKind = "object"
	Map     ParamKind = "map"
	Array   ParamKind = "array"
)

type ParsedParam struct {
	Name        string
	SampleValue string
	Optional    bool
	Kind        ParamKind
	SubKind     ParamKind
	Message     ParsedMessage
}

var (
	visited map[string]struct{}
	parsed  map[string]ParsedMessage
)

func isParsed(name string) bool {
	_, ok := parsed[name]

	return ok
}

func isVisited(name string) bool {
	_, ok := visited[name]

	return ok
}

func Parse(desc ServiceDesc) ParsedService {
	return ParseService(desc.Desc())
}

func ParseService(svc *Service) ParsedService {
	// reset the parsed map
	// we need this map, to prevent infinite recursion
	parsed = make(map[string]ParsedMessage)
	visited = make(map[string]struct{})

	pd := ParsedService{
		Origin: svc,
	}

	for _, c := range svc.Contracts {
		pd.Contracts = append(pd.Contracts, parseContract(c)...)
	}

	return pd
}

func parseContract(c Contract) []ParsedContract {
	var pcs []ParsedContract
	for _, s := range c.RouteSelectors {
		pc := ParsedContract{
			GroupName: c.Name,
			Name:      s.Name,
			Encoding:  s.Selector.GetEncoding().Tag(),
		}

		switch r := s.Selector.(type) {
		case kit.RESTRouteSelector:
			pc.Type = REST
			pc.Path = r.GetPath()
			pc.Method = r.GetMethod()
		case kit.RPCRouteSelector:
			pc.Type = RPC
			pc.Predicate = r.GetPredicate()
		}

		pc.Request = ParsedRequest{
			Message: parseMessage(c.Input, s.Selector.GetEncoding()),
		}

		pc.Responses = append(
			pc.Responses,
			ParsedResponse{
				Message: parseMessage(c.Output, s.Selector.GetEncoding()),
			},
		)

		for _, e := range c.PossibleErrors {
			pc.Responses = append(
				pc.Responses,
				ParsedResponse{
					Message: parseMessage(e.Message, s.Selector.GetEncoding()),
					ErrCode: e.Code,
					ErrItem: e.Item,
				},
			)
		}

		pcs = append(pcs, pc)
	}

	return pcs
}

func parseMessage(m kit.Message, enc kit.Encoding) ParsedMessage {
	mt := reflect.TypeOf(m)
	pm := ParsedMessage{
		Name: mt.Name(),
	}

	if mt.Kind() == reflect.Ptr {
		mt = mt.Elem()
	}

	visited[mt.Name()] = struct{}{}

	if mt.Kind() != reflect.Struct {
		return pm
	}

	// if we are here, it means that mt is a struct
	var params []ParsedParam
	for i := 0; i < mt.NumField(); i++ {
		f := mt.Field(i)
		ft := f.Type
		pp := ParsedParam{
			Name: f.Tag.Get(enc.Tag()),
		}

		if ft.Kind() == reflect.Ptr {
			pp.Optional = true
			ft = ft.Elem()
		}
		pp.Kind = parseKind(ft.Kind())

		switch ft.Kind() {
		case reflect.Map:
			if ft.Key().Kind() != reflect.String {
				continue
			}

			fallthrough
		case reflect.Array, reflect.Slice:
			pp.SubKind = parseKind(ft.Elem().Kind())
			if pp.SubKind != Object {
				break
			}
			ft = ft.Elem()

			fallthrough
		case reflect.Struct:
			if isParsed(ft.Name()) {
				pp.Message = parsed[ft.Name()]
			} else if isVisited(ft.Name()) {
				panic("infinite recursion detected")
			} else {
				fmt.Println(ft.Name())
				pp.Message = parseMessage(reflect.New(ft).Interface(), enc)
			}
		case reflect.Chan, reflect.Interface, reflect.Func:
			continue
		}

		params = append(params, pp)
	}

	pm.Params = params
	parsed[mt.Name()] = pm

	return pm
}

func parseKind(k reflect.Kind) ParamKind {
	switch k {
	case reflect.String:
		return String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Integer
	case reflect.Float32, reflect.Float64:
		return Float
	case reflect.Struct:
		return Object
	case reflect.Map:
		return Map
	case reflect.Slice, reflect.Array:
		return Array
	}

	return ""
}
