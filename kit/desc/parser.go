package desc

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/goccy/go-json"
)

type ParsedService struct {
	// Origin is the original service descriptor untouched by the parser
	Origin *Service
	// Contracts is the list of parsed contracts. The relation between ParsedContract
	// and Contract is not 1:1 because a Contract can have multiple RouteSelectors.
	// Each RouteSelector will be parsed into a ParsedContract.
	Contracts []ParsedContract

	// internals
	visited map[string]struct{}
	parsed  map[string]ParsedMessage
}

func (ps *ParsedService) Messages() []ParsedMessage {
	var msgs []ParsedMessage //nolint:prealloc
	for _, m := range ps.parsed {
		msgs = append(msgs, m)
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Name < msgs[j].Name
	})

	return msgs
}

func (ps *ParsedService) parseContract(c Contract) []ParsedContract {
	var pcs []ParsedContract //nolint:prealloc
	for idx, s := range c.RouteSelectors {
		name := s.Name
		if name == "" {
			name = c.Name
		}
		pc := ParsedContract{
			Index:     idx,
			GroupName: c.Name,
			Name:      name,
			Encoding:  s.Selector.GetEncoding().Tag(),
		}

		switch r := s.Selector.(type) {
		case kit.RESTRouteSelector:
			pc.Type = REST
			pc.Path = r.GetPath()
			pc.Method = r.GetMethod()

			for _, p := range strings.Split(pc.Path, "/") {
				if strings.HasPrefix(p, ":") {
					pc.PathParams = append(pc.PathParams, p[1:])
				}
			}
		case kit.RPCRouteSelector:
			pc.Type = RPC
			pc.Predicate = r.GetPredicate()
		}

		pc.Request = ParsedRequest{
			Message: ps.parseMessage(c.Input, s.Selector.GetEncoding()),
		}

		if c.Output != nil {
			pc.Responses = append(
				pc.Responses,
				ParsedResponse{
					Message: ps.parseMessage(c.Output, s.Selector.GetEncoding()),
				},
			)
		}

		for _, e := range c.PossibleErrors {
			pc.Responses = append(
				pc.Responses,
				ParsedResponse{
					Message: ps.parseMessage(e.Message, s.Selector.GetEncoding()),
					ErrCode: e.Code,
					ErrItem: e.Item,
				},
			)
		}

		pcs = append(pcs, pc)
	}

	return pcs
}

func (ps *ParsedService) parseMessage(m kit.Message, enc kit.Encoding) ParsedMessage {
	mt := reflect.TypeOf(m)
	pm := ParsedMessage{
		Name: mt.Name(),
	}

	if mt.Kind() == reflect.Ptr {
		mt = mt.Elem()
	}

	ps.visited[mt.Name()] = struct{}{}

	if mt.Kind() != reflect.Struct {
		return pm
	}

	tagName := enc.Tag()
	if tagName == "" {
		tagName = kit.JSON.Tag()
	}

	// if we are here, it means that mt is a struct
	var params []ParsedParam
	for i := 0; i < mt.NumField(); i++ {
		f := mt.Field(i)
		ft := f.Type
		ptn := getParsedStructTag(f.Tag, tagName)
		pp := ParsedParam{
			Name: ptn.Name,
			Tag:  ptn,
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
			if pp.SubKind != Object && pp.SubKind != Array && pp.SubKind != Map {
				break
			}
			ft = ft.Elem()

			fallthrough
		case reflect.Struct:
			if ps.isParsed(ft.Name()) {
				pp.Message = ps.parsed[ft.Name()]
			} else if ps.isVisited(ft.Name()) {
				panic("infinite recursion detected")
			} else {
				pp.Message = ps.parseMessage(reflect.New(ft).Interface(), enc)
			}
		case reflect.Chan, reflect.Interface, reflect.Func:
			continue
		}

		params = append(params, pp)
	}

	pm.Params = params
	ps.parsed[mt.Name()] = pm

	return pm
}

func (ps *ParsedService) isParsed(name string) bool {
	_, ok := ps.parsed[name]

	return ok
}

func (ps *ParsedService) isVisited(name string) bool {
	_, ok := ps.visited[name]

	return ok
}

type ContractType string

const (
	REST ContractType = "REST"
	RPC  ContractType = "RPC"
)

type ParsedContract struct {
	Index     int
	GroupName string
	Name      string
	Encoding  string

	Type       ContractType
	Path       string
	PathParams []string
	Method     string
	Predicate  string

	Request   ParsedRequest
	Responses []ParsedResponse
}

func (pc ParsedContract) SuggestName() string {
	if pc.Name != "" {
		return pc.Name
	}

	switch pc.Type {
	case REST:
		parts := strings.Split(pc.Path, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if strings.HasPrefix(parts[i], ":") {
				continue
			}

			return utils.ToCamel(parts[i])
		}
	case RPC:
		return utils.ToCamel(pc.Predicate)
	}

	return fmt.Sprintf("%s%d", pc.GroupName, pc.Index)
}

func (pc ParsedContract) OKResponse() ParsedResponse {
	for _, r := range pc.Responses {
		if !r.IsError() {
			return r
		}
	}

	return ParsedResponse{}
}

func (pc ParsedContract) IsPathParam(name string) bool {
	for _, p := range pc.PathParams {
		if p == name {
			return true
		}
	}

	return false
}

type ParsedMessage struct {
	Name   string
	Params []ParsedParam
}

func (pm ParsedMessage) JSON() string {
	m := map[string]interface{}{}
	for _, p := range pm.Params {
		switch p.Kind {
		case Map:
			m[p.Name] = map[string]interface{}{}
		case Array:
			m[p.Name] = []interface{}{}
		case Integer, Float:
			m[p.Name] = 0
		default:
			m[p.Name] = p.Kind
		}
	}

	d, _ := json.MarshalIndent(m, "", "  ")

	return string(d)
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
	Bool    ParamKind = "boolean"
	String  ParamKind = "string"
	Integer ParamKind = "integer"
	Float   ParamKind = "float"
	Object  ParamKind = "object"
	Map     ParamKind = "map"
	Array   ParamKind = "array"
)

type ParsedParam struct {
	Name        string
	Tag         ParsedStructTag
	SampleValue string
	Optional    bool
	Kind        ParamKind
	// SubKind is the kind of the elements of the array or map. For other kinds,
	// it is empty.
	SubKind ParamKind
	// Message is the parsed message if the kind is Object, Map or Array.
	// In Map and Array, the Message is the parsed message of the elements.
	// In Object, the Message is the parsed message of the struct.
	Message ParsedMessage
}

// Parse extracts the Service descriptor from the input ServiceDesc
// Refer to ParseService for more details.
func Parse(desc ServiceDesc) ParsedService {
	return ParseService(desc.Desc())
}

// ParseService extracts information from a Service descriptor using reflection.
// It returns a ParsedService. The ParsedService is useful to generate custom
// code based on the service descriptor.
// In the contrib package this is used to generate the swagger spec and postman collections.
func ParseService(svc *Service) ParsedService {
	// reset the parsed map
	// we need this map, to prevent infinite recursion

	pd := ParsedService{
		Origin:  svc,
		parsed:  make(map[string]ParsedMessage),
		visited: make(map[string]struct{}),
	}

	for _, c := range svc.Contracts {
		pd.Contracts = append(pd.Contracts, pd.parseContract(c)...)
	}

	return pd
}

func parseKind(k reflect.Kind) ParamKind {
	switch k {
	case reflect.Bool:
		return Bool
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

const (
	swagTagKey   = "swag"
	swagSep      = ";"
	swagIdentSep = ":"
	swagValueSep = ","
)

type ParsedStructTag struct {
	Name           string
	Optional       bool
	PossibleValues []string
}

func getParsedStructTag(tag reflect.StructTag, name string) ParsedStructTag {
	pst := ParsedStructTag{}
	nameTag := tag.Get(name)
	if nameTag == "" {
		return pst
	}

	// This is a hack to remove omitempty from tags
	fNameParts := strings.Split(nameTag, swagValueSep)
	if len(fNameParts) > 0 {
		pst.Name = strings.TrimSpace(fNameParts[0])
	}

	swagTag := tag.Get(swagTagKey)
	parts := strings.Split(swagTag, swagSep)
	for _, p := range parts {
		x := strings.TrimSpace(strings.ToLower(p))
		switch {
		case x == "optional":
			pst.Optional = true
		case strings.HasPrefix(x, "enum:"):
			xx := strings.SplitN(p, swagIdentSep, 2)
			if len(xx) == 2 {
				xx = strings.Split(xx[1], swagValueSep)
				for _, v := range xx {
					pst.PossibleValues = append(pst.PossibleValues, strings.TrimSpace(v))
				}
			}
		}
	}

	return pst
}
