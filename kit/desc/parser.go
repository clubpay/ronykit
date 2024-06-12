package desc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
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
	parsed  map[string]*ParsedMessage
}

func (ps *ParsedService) Messages() []ParsedMessage {
	var msgs []ParsedMessage //nolint:prealloc
	for _, m := range ps.parsed {
		msgs = append(msgs, *m)
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
				if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
					pc.PathParams = append(pc.PathParams, p[1:len(p)-1])
				}
			}
		case kit.RPCRouteSelector:
			pc.Type = RPC
			pc.Predicate = r.GetPredicate()
		}

		pc.Request = ParsedRequest{
			Headers: c.InputHeaders,
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
	if mt.Kind() == reflect.Ptr {
		mt = mt.Elem()
	}

	pm := ParsedMessage{
		original: m,
		Name:     mt.Name(),
		Kind:     parseKind(mt),
		RKind:    mt.Kind(),
		Type:     typ("", mt),
		RType:    mt,
	}

	switch {
	case mt == reflect.TypeOf(kit.RawMessage{}):
		return pm
	case mt == reflect.TypeOf(kit.MultipartFormMessage{}):
		return pm
	case mt.Kind() != reflect.Struct:
		return pm
	}

	ps.visited[mt.Name()] = struct{}{}

	tagName := enc.Tag()
	if tagName == "" {
		tagName = kit.JSON.Tag()
	}

	// if we are here, it means that mt is a struct
	var fields []ParsedField
	for i := 0; i < mt.NumField(); i++ {
		f := mt.Field(i)
		ft := f.Type
		var optional bool
		if ft.Kind() == reflect.Ptr {
			optional = true
			ft = ft.Elem()
		}

		ptn := getParsedStructTag(f.Tag, tagName)

		fields = append(
			fields,
			ParsedField{
				GoName:   f.Name,
				Name:     ptn.Name,
				Tag:      ptn,
				Optional: optional,
				Embedded: f.Anonymous,
				Element:  utils.ValPtr(ps.parseElement(ft, enc)),
			},
		)
	}

	pm.Fields = fields
	ps.parsed[mt.Name()] = &pm

	return pm
}

func (ps *ParsedService) parseElement(ft reflect.Type, enc kit.Encoding) ParsedElement {
	kind := parseKind(ft)
	pe := ParsedElement{
		Kind:  kind,
		RKind: ft.Kind(),
		Type:  typ("", ft),
		RType: ft,
	}
	switch kind {
	case Map:
		// we only support maps with string keys
		pe.Key = utils.ValPtr(ps.parseElement(ft.Key(), enc))
		pe.Element = utils.ValPtr(ps.parseElement(ft.Elem(), enc))

	case Array:
		pe.Element = utils.ValPtr(ps.parseElement(ft.Elem(), enc))

	case Object:
		if ps.isParsed(ft.Name()) {
			pe.Message = ps.parsed[ft.Name()]
		} else if ps.isVisited(ft.Name()) {
			panic(fmt.Sprintf("infinite recursion detected: %s", ft.Name()))
		} else {
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			pe.Message = utils.ValPtr(ps.parseMessage(reflect.New(ft).Interface(), enc))
		}
	}

	return pe
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

type ParsedRequest struct {
	Headers []Header
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

type Kind string

const (
	None                    Kind = ""
	Bool                    Kind = "boolean"
	String                  Kind = "string"
	Integer                 Kind = "integer"
	Float                   Kind = "float"
	Byte                    Kind = "byte"
	Object                  Kind = "object"
	Map                     Kind = "map"
	Array                   Kind = "array"
	kitRawMessage           Kind = "kitRawMessage"
	kitMultipartFormMessage Kind = "kitMultipartFormMessage"
)

type ParsedMessage struct {
	original kit.Message
	Name     string
	Kind     Kind
	RKind    reflect.Kind
	Type     string
	RType    reflect.Type
	Fields   []ParsedField
}

func (pm ParsedMessage) IsSpecial() bool {
	return pm.Kind == kitRawMessage || pm.Kind == kitMultipartFormMessage
}

func (pm ParsedMessage) JSON() string {
	mJSON, _ := json.MarshalIndent(pm, "", "  ")

	return utils.B2S(mJSON)
}

func (pm ParsedMessage) String() string {
	sb := strings.Builder{}
	sb.WriteString(pm.Name)
	sb.WriteString("[")
	for idx, p := range pm.Fields {
		if idx > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name)
		sb.WriteString(":")
		sb.WriteString(string(p.Element.Kind))

		switch p.Element.Kind {
		case Map, Array:
			sb.WriteString(":")
			sb.WriteString(p.Element.String())
		case Object:
			sb.WriteString(":")
		}
	}
	sb.WriteString("]")

	return sb.String()
}

func (pm ParsedMessage) FieldByName(name string) *ParsedField {
	for _, f := range pm.Fields {
		if f.Name == name {
			return &f
		}
	}

	return nil
}

func (pm ParsedMessage) FieldByGoName(name string) *ParsedField {
	for _, f := range pm.Fields {
		if f.GoName == name {
			return &f
		}
	}

	return nil
}

type ParsedField struct {
	GoName      string
	Name        string
	Tag         ParsedStructTag
	SampleValue string
	Optional    bool
	Embedded    bool

	Element *ParsedElement
}

type ParsedElement struct {
	Kind  Kind
	RKind reflect.Kind
	Type  string
	RType reflect.Type

	// Message is the parsed message if the kind is Object.
	Message *ParsedMessage // only if Kind == Object
	Element *ParsedElement // if Kind == Array ||  Map
	Key     *ParsedElement // only if Kind == Map
}

func (pf ParsedElement) String() string {
	switch pf.Kind {
	case Map:
		return fmt.Sprintf("map[%s]", pf.Element.String())
	case Array:
		return fmt.Sprintf("array[%s]", pf.Element.String())
	case Object:
		return pf.Message.String()
	default:
		return string(pf.Kind)
	}
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
		parsed:  make(map[string]*ParsedMessage),
		visited: make(map[string]struct{}),
	}

	for _, c := range svc.Contracts {
		pd.Contracts = append(pd.Contracts, pd.parseContract(c)...)
	}

	return pd
}

func parseKind(t reflect.Type) Kind {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle special messages
	switch t {
	default:
	case reflect.TypeOf(kit.MultipartFormMessage{}):
		return kitMultipartFormMessage
	case reflect.TypeOf(kit.RawMessage{}):
		return kitRawMessage
	}

	switch t.Kind() {
	default:
	case reflect.Bool:
		return Bool
	case reflect.String:
		return String
	case reflect.Uint8, reflect.Int8:
		return Byte
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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

	return None
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
	Deprecated     bool
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
		case x == "deprecated":
			pst.Deprecated = true
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
