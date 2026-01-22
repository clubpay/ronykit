package desc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
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

func (ps *ParsedService) MessageByName(name string) *ParsedMessage {
	for _, m := range ps.parsed {
		if m.Name == name {
			return m
		}
	}

	return nil
}

func (ps *ParsedService) parseContract(c Contract) []ParsedContract {
	var pcs []ParsedContract //nolint:prealloc

	for idx, s := range c.RouteSelectors {
		pc := ParsedContract{
			Index:        idx,
			GroupName:    c.Name,
			Name:         utils.Coalesce(s.Name, c.Name),
			SelectorName: utils.Coalesce(s.Name, s.Selector.String()),
			Deprecated:   s.Deprecated,
			Encoding:     s.Selector.GetEncoding().Tag(),
		}

		switch r := s.Selector.(type) {
		case kit.RESTRouteSelector:
			pc.Type = REST
			pc.Path = r.GetPath()
			pc.Method = r.GetMethod()

			for p := range strings.SplitSeq(pc.Path, "/") {
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
			Message: ps.parseMessage(c.Input, c.InputMeta, s.Selector.GetEncoding()),
		}

		if c.Output != nil {
			pc.Responses = append(
				pc.Responses,
				ParsedResponse{
					Message: ps.parseMessage(c.Output, c.OutputMeta, s.Selector.GetEncoding()),
				},
			)
		}

		for _, e := range c.PossibleErrors {
			pc.Responses = append(
				pc.Responses,
				ParsedResponse{
					Message: ps.parseMessage(e.Message, e.Meta, s.Selector.GetEncoding()),
					ErrCode: e.Code,
					ErrItem: e.Item,
				},
			)
		}

		if c.DefaultError != nil {
			pc.DefaultError = &ParsedResponse{
				Message: ps.parseMessage(c.DefaultError.Message, c.DefaultError.Meta, s.Selector.GetEncoding()),
				ErrCode: c.DefaultError.Code,
				ErrItem: c.DefaultError.Item,
			}
		}

		pcs = append(pcs, pc)
	}

	return pcs
}

func (ps *ParsedService) parseMessage(m kit.Message, meta MessageMeta, enc kit.Encoding) ParsedMessage {
	mt := reflect.TypeOf(m)
	if mt.Kind() == reflect.Pointer {
		mt = mt.Elem()
	}

	pm := ParsedMessage{
		original:       m,
		Name:           mt.Name(),
		PkgPath:        mt.PkgPath(),
		Kind:           parseKind(mt),
		RKind:          mt.Kind(),
		Type:           typ("", mt),
		RType:          mt,
		ImplementError: mt.Implements(reflect.TypeFor[kit.ErrorMessage]()),
		Meta:           meta,
	}

	switch {
	case mt == reflect.TypeFor[kit.RawMessage]():
		return pm
	case mt == reflect.TypeFor[kit.MultipartFormMessage]():
		return pm
	case mt.Kind() != reflect.Struct:
		return pm
	}

	ps.setVisited(mt)

	tagName := enc.Tag()
	if tagName == "" {
		tagName = kit.JSON.Tag()
	}

	// if we are here, it means that mt is a struct
	fields := make([]ParsedField, 0, mt.NumField())
	for i := range mt.NumField() {
		f := mt.Field(i)
		ft := f.Type
		ptn := getParsedStructTag(f.Tag, tagName)

		fields = append(
			fields,
			ParsedField{
				GoName:   f.Name,
				Name:     ptn.Value,
				Tag:      ptn,
				Optional: ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice || ft.Kind() == reflect.Map,
				Embedded: f.Anonymous,
				Element:  utils.ValPtr(ps.parseElement(ft, enc)),
				Exported: f.IsExported(),
				Meta:     meta.Fields[f.Name],
			},
		)
	}

	pm.Fields = fields
	ps.setParsed(mt, &pm)

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
		pe.Key = utils.ValPtr(ps.parseElement(ft.Key(), enc))
		pe.Element = utils.ValPtr(ps.parseElement(ft.Elem(), enc))

	case Array:
		pe.Element = utils.ValPtr(ps.parseElement(ft.Elem(), enc))

	case Object:
		switch {
		default:
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			pe.Message = utils.ValPtr(ps.parseMessage(reflect.New(ft).Interface(), MessageMeta{}, enc))
		case ps.isParsed(ft):
			pe.Message = ps.getParsed(ft)
		case ps.isVisited(ft):
			panic(fmt.Sprintf("infinite recursion detected: %s", ft.Name()))
		}
	}

	return pe
}

func (ps *ParsedService) setParsed(t reflect.Type, pm *ParsedMessage) {
	ps.parsed[t.PkgPath()+t.Name()] = pm
}

func (ps *ParsedService) getParsed(t reflect.Type) *ParsedMessage {
	return ps.parsed[t.PkgPath()+t.Name()]
}

func (ps *ParsedService) isParsed(t reflect.Type) bool {
	_, ok := ps.parsed[t.PkgPath()+t.Name()]

	return ok
}

func (ps *ParsedService) setVisited(t reflect.Type) {
	ps.visited[t.PkgPath()+t.Name()] = struct{}{}
}

func (ps *ParsedService) isVisited(t reflect.Type) bool {
	_, ok := ps.visited[t.PkgPath()+t.Name()]

	return ok
}

type ContractType string

const (
	REST ContractType = "REST"
	RPC  ContractType = "RPC"
)

type ParsedContract struct {
	Index        int
	GroupName    string
	Name         string
	SelectorName string
	Encoding     string
	Deprecated   bool

	Type       ContractType
	Path       string
	PathParams []string
	Method     string
	Predicate  string

	Request      ParsedRequest
	Responses    []ParsedResponse
	DefaultError *ParsedResponse
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
	return slices.Contains(pc.PathParams, name)
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
	KitRawMessage           Kind = "kitRawMessage"
	KitMultipartFormMessage Kind = "kitMultipartFormMessage"
)

type ParsedMessage struct {
	original       kit.Message
	Name           string
	PkgPath        string
	Kind           Kind
	RKind          reflect.Kind
	Type           string
	RType          reflect.Type
	Fields         []ParsedField
	ImplementError bool
	Meta           MessageMeta
}

func (pm ParsedMessage) IsSpecial() bool {
	return pm.Kind == KitRawMessage || pm.Kind == KitMultipartFormMessage
}

func (pm ParsedMessage) GoName() string {
	if pm.IsSpecial() {
		switch pm.Kind {
		case KitRawMessage:
			return "kit.RawMessage"
		case KitMultipartFormMessage:
			return "kit.MultipartFormMessage"
		}
	}

	return pm.Name
}

func (pm ParsedMessage) JSON() string {
	mJSON, _ := json.MarshalIndent(pm.original, "", "  ")

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

func (pm ParsedMessage) CodeField() string {
	var fn string

	for _, f := range pm.Fields {
		x := strings.ToLower(f.GoName)
		if f.Element.Type != "int" {
			continue
		}

		if x == "code" {
			return f.GoName
		}

		if strings.HasPrefix(f.GoName, "code") {
			fn = f.GoName
		}
	}

	return fn
}

func (pm ParsedMessage) ItemField() string {
	var fn string

	for _, f := range pm.Fields {
		x := strings.ToLower(f.GoName)
		if f.Element.RType.Kind() != reflect.String {
			continue
		}

		if x == "item" || x == "items" {
			return f.GoName
		}

		if strings.HasPrefix(f.GoName, "item") {
			fn = f.GoName
		}
	}

	return fn
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

func (pm ParsedMessage) ExportedFields() []*ParsedField {
	var fields []*ParsedField

	for idx := range pm.Fields {
		if pm.Fields[idx].Exported {
			fields = append(fields, &pm.Fields[idx])
		}
	}

	return fields
}

func (pm ParsedMessage) TotalExportedFields() int {
	var count int

	for _, f := range pm.Fields {
		if f.Exported {
			count++
		}
	}

	return count
}

type ParsedField struct {
	GoName      string
	Name        string
	Tag         ParsedStructTag
	SampleValue string
	Optional    bool
	Embedded    bool
	Exported    bool
	Meta        FieldMeta

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
	// we need this map to prevent infinite recursion
	pd := ParsedService{
		Origin:  svc,
		parsed:  make(map[string]*ParsedMessage),
		visited: make(map[string]struct{}),
	}

	for _, c := range svc.Contracts {
		if c.DefaultError == nil {
			c.DefaultError = svc.DefaultError
		}

		c.PossibleErrors = append(c.PossibleErrors, svc.PossibleErrors...)
		pd.Contracts = append(pd.Contracts, pd.parseContract(c)...)
	}

	return pd
}

func parseKind(t reflect.Type) Kind {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Handle special messages
	switch t {
	default:
	case reflect.TypeFor[kit.MultipartFormMessage]():
		return KitMultipartFormMessage
	case reflect.TypeFor[kit.RawMessage]():
		return KitRawMessage
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
		// Check if type implements String() string method
		if t.Implements(reflect.TypeFor[fmt.Stringer]()) {
			return String
		}

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
	Raw            reflect.StructTag
	Name           string
	Value          string
	Optional       bool
	PossibleValues []string
	Deprecated     bool
	OmitEmpty      bool
	OmitZero       bool
}

func (pst ParsedStructTag) Tags(keys ...string) map[string]string {
	tags := make(map[string]string)

	for _, k := range keys {
		v, ok := pst.Raw.Lookup(k)
		if ok {
			tags[k] = v
		}
	}

	return tags
}

func (pst ParsedStructTag) Get(key string) string {
	return pst.Raw.Get(key)
}

func getParsedStructTag(tag reflect.StructTag, name string) ParsedStructTag {
	pst := ParsedStructTag{
		Raw:  tag,
		Name: name,
	}

	nameTag := tag.Get(name)
	if nameTag == "" {
		return pst
	}

	// This is a hack to remove omitempty from tags
	if fNameParts := strings.Split(nameTag, swagValueSep); len(fNameParts) > 0 {
		pst.Value = strings.TrimSpace(fNameParts[0])

		if len(fNameParts) > 1 {
			for _, v := range fNameParts[1:] {
				switch v {
				case "omitempty":
					pst.OmitEmpty = true
				case "omitzero":
					pst.OmitZero = true
				}
			}
		}
	}

	swagTag := tag.Get(swagTagKey)

	parts := strings.SplitSeq(swagTag, swagSep)
	for p := range parts {
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
