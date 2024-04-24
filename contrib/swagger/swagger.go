package swagger

import (
	"container/list"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/go-openapi/spec"
	"github.com/rbretecher/go-postman-collection"
)

const kitRawMessage = "RawMessage"

type Generator struct {
	tagName string
	title   string
	version string
	desc    string
}

func New(title, ver, desc string) *Generator {
	sg := &Generator{
		title:   title,
		version: ver,
		desc:    desc,
	}

	return sg
}

func (sg *Generator) WithTag(tagName string) *Generator {
	sg.tagName = tagName

	return sg
}

func (sg *Generator) WriteSwagToFile(filename string, services ...desc.ServiceDesc) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	return sg.WriteSwagTo(f, services...)
}

func (sg *Generator) WriteSwagTo(w io.Writer, descs ...desc.ServiceDesc) error {
	swag := &spec.Swagger{}
	swag.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Description: sg.desc,
			Title:       sg.title,
			Version:     sg.version,
		},
	}
	swag.Schemes = []string{"http", "https"}
	swag.Swagger = "2.0"

	for _, d := range descs {
		ps := desc.Parse(d)

		swag.Tags = append(
			swag.Tags,
			spec.NewTag(ps.Origin.Name, ps.Origin.Description, nil),
		)

		for _, m := range ps.Messages() {
			addSwagDefinition(swag, m)
		}

		for _, c := range ps.Contracts {
			addSwagOp(swag, ps.Origin.Name, c)
		}
	}

	swaggerJSON, err := swag.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.Write(swaggerJSON)

	return err
}

//nolint:cyclop
func addSwagOp(swag *spec.Swagger, serviceName string, c desc.ParsedContract) {
	if swag.Paths == nil {
		swag.Paths = &spec.Paths{
			Paths: map[string]spec.PathItem{},
		}
	}
	var contentType string
	switch c.Encoding {
	case kit.JSON.Tag():
		contentType = "application/json"
	case kit.Proto.Tag():
		contentType = "application/x-protobuf"
	case kit.MSG.Tag():
		contentType = "application/octet-stream"
	default:
		contentType = "application/json"
	}

	if c.Type != desc.REST {
		return
	}

	opID := c.Name
	op := spec.NewOperation(opID).
		WithTags(serviceName).
		WithProduces(contentType).
		WithConsumes(contentType)
	if c.OKResponse().Message.Name == kitRawMessage {
		op.RespondsWith(
			http.StatusOK,
			spec.NewResponse(),
		)
	} else {
		op.RespondsWith(
			http.StatusOK,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.OKResponse().Message.Name)),
				),
		)
	}

	possibleErrors := map[int][]string{}
	for _, r := range c.Responses {
		if !r.IsError() {
			continue
		}

		possibleErrors[r.ErrCode] = append(possibleErrors[r.ErrCode], r.ErrItem)
		op.RespondsWith(
			r.ErrCode,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", r.Message.Name)),
				).
				WithDescription(fmt.Sprintf("Items: %s", strings.Join(possibleErrors[r.ErrCode], ", "))),
		)
	}

	setSwagInput(op, c)

	restPath := fixPathForSwag(c.Path)
	pathItem := swag.Paths.Paths[restPath]
	switch strings.ToUpper(c.Method) {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodDelete:
		pathItem.Delete = op
	case http.MethodPost:
		if c.Request.Message.Name != kitRawMessage {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
				),
			)
		}
		pathItem.Post = op
	case http.MethodPut:
		if c.Request.Message.Name != kitRawMessage {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
				),
			)
		}
		pathItem.Put = op
	case http.MethodPatch:
		if c.Request.Message.Name != kitRawMessage {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
				),
			)
		}
		pathItem.Patch = op
	}
	swag.Paths.Paths[restPath] = pathItem
}

func setSwagInput(op *spec.Operation, c desc.ParsedContract) {
	if len(c.Request.Message.Fields) == 0 {
		return
	}

	for _, p := range c.Request.Message.Fields {
		if c.IsPathParam(p.Name) {
			op.AddParam(
				setSwaggerParam(spec.PathParam(p.Name), p),
			)
		} else if c.Method == http.MethodGet || c.Method == http.MethodDelete {
			op.AddParam(
				setSwaggerParam(spec.QueryParam(p.Name), p),
			)
		}
	}

	for _, hdr := range c.Request.Headers {
		if hdr.Required {
			op.AddParam(
				spec.HeaderParam(hdr.Name).
					AsRequired(),
			)
		} else {
			op.AddParam(
				spec.HeaderParam(hdr.Name).
					AsOptional(),
			)
		}
	}
}

func addSwagDefinition(swag *spec.Swagger, m desc.ParsedMessage) {
	if swag.Definitions == nil {
		swag.Definitions = map[string]spec.Schema{}
	}

	swag.Definitions[m.Name] = toSwagDefinition(m)
}

func toSwagDefinition(m desc.ParsedMessage) spec.Schema {
	def := spec.Schema{}
	def.Typed("object", "")

	fields := list.New()
	for _, f := range m.Fields {
		fields.PushFront(f)
	}

	for fields.Back() != nil {
		p := fields.Remove(fields.Back()).(desc.ParsedField) //nolint:errcheck,forcetypeassert

		name, kind, wrapFuncChain := getWrapFunc(p)

		switch kind {
		default:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.StringProperty()))
		case desc.Object:
			if p.Embedded {
				for _, f := range p.Message.Fields {
					fields.PushBack(f)
				}
			} else {
				def.SetProperty(p.Name, wrapFuncChain.Apply(spec.RefProperty(fmt.Sprintf("#/definitions/%s", name))))
			}
		case desc.String:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.StringProperty()))
		case desc.Float:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.Float64Property()))
		case desc.Integer:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.Int64Property()))
		case desc.Bool:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.BoolProperty()))
		case desc.Byte:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.Int8Property()))
		}
	}

	return def
}

func getWrapFunc(p desc.ParsedField) (string, desc.Kind, schemaWrapperChain) {
	var (
		wrapFuncChain = schemaWrapperChain{}
		name          string
	)

	msg := p.Message
	kind := p.Kind
	elem := p.Element
Loop:
	switch kind {
	default:
	case desc.Object:
		name = msg.Name
	case desc.Map:
		wrapFuncChain = wrapFuncChain.Add(spec.MapProperty)
		kind = elem.Kind
		msg = elem.Message
		elem = elem.Element

		goto Loop
	case desc.Array:
		wrapFuncChain = wrapFuncChain.Add(spec.ArrayProperty)
		kind = elem.Kind
		msg = elem.Message
		elem = elem.Element

		goto Loop
	}

	enum := utils.Map(
		func(v string) any { return v },
		p.Tag.PossibleValues,
	)
	if len(enum) > 0 {
		wrapFuncChain = wrapFuncChain.Add(
			func(schema *spec.Schema) *spec.Schema {
				schema.Enum = enum

				return schema
			},
		)
	}

	if p.Tag.Optional || p.Optional {
		wrapFuncChain = wrapFuncChain.Add(
			func(schema *spec.Schema) *spec.Schema {
				spacer := ""
				if len(schema.Description) > 0 {
					spacer = " "
				}
				schema.Description = fmt.Sprintf("[Optional]%s%s", spacer, schema.Description)

				return schema
			},
		)
	}

	if p.Tag.Deprecated {
		wrapFuncChain = wrapFuncChain.Add(
			func(schema *spec.Schema) *spec.Schema {
				spacer := ""
				if len(schema.Description) > 0 {
					spacer = " "
				}
				schema.Description = fmt.Sprintf("[Deprecated]%s%s", spacer, schema.Description)

				return schema
			},
		)
	}

	return name, kind, wrapFuncChain
}

func (sg *Generator) WritePostmanToFile(filename string, services ...desc.ServiceDesc) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	return sg.WritePostmanTo(f, services...)
}

func (sg *Generator) WritePostmanTo(w io.Writer, descs ...desc.ServiceDesc) error {
	col := postman.CreateCollection(sg.title, sg.desc)
	col.Variables = append(
		col.Variables,
		&postman.Variable{
			Type: "string",
			Name: "baseURL",
			Key:  "baseURL",
		},
	)

	for _, d := range descs {
		ps := desc.Parse(d)

		colItems := col.AddItemGroup(ps.Origin.Name)

		for _, c := range ps.Contracts {
			switch c.Type {
			case desc.REST:
				colItems.AddItem(toPostmanItem(c))
			}
		}
	}

	return col.Write(w, postman.V210)
}

func toPostmanItem(c desc.ParsedContract) *postman.Items {
	itm := postman.CreateItem(
		postman.Item{
			Name:                    c.SuggestName(),
			Variables:               nil,
			Events:                  nil,
			ProtocolProfileBehavior: nil,
		},
	)
	if c.Encoding == "" {
		c.Encoding = "json"
	}

	for _, pp := range c.PathParams {
		v := &postman.Variable{
			Name: pp,
			Key:  pp,
		}
		for _, p := range c.Request.Message.Fields {
			if p.Name == pp {
				v.Type = string(p.Kind)
				v.Value = p.SampleValue

				break
			}
		}
		itm.Variables = append(itm.Variables, v)
	}

	var queryParams []*postman.QueryParam
	if len(c.PathParams) > len(c.Request.Message.Fields) || c.Method == "GET" {
		for _, p := range c.Request.Message.Fields {
			found := false
			for _, pp := range c.PathParams {
				if p.Name == pp {
					found = true

					break
				}
			}

			if !found {
				queryParams = append(
					queryParams,
					&postman.QueryParam{
						Key:   p.Name,
						Value: p.SampleValue,
					},
				)
			}
		}
	}
	itm.Request = &postman.Request{
		URL: &postman.URL{
			Raw: fmt.Sprintf("{{baseURL}}%s", c.Path),
			Host: []string{
				"{{baseURL}}",
			},
			Path:  strings.Split(c.Path, "/"),
			Query: queryParams,
		},
		Method: postman.Method(c.Method),
		Body: &postman.Body{
			Mode: "raw",
			Raw:  c.Request.Message.JSON(),
			Options: &postman.BodyOptions{
				Raw: postman.BodyOptionsRaw{
					Language: c.Encoding,
				},
			},
		},
	}

	for _, hdr := range c.Request.Headers {
		itm.Request.Header = append(
			itm.Request.Header,
			&postman.Header{
				Key:   hdr.Name,
				Value: "",
			},
		)
	}

	return itm
}

func setSwaggerParam(p *spec.Parameter, pp desc.ParsedField) *spec.Parameter {
	if pp.Tag.Optional {
		p.AsOptional()
	} else {
		p.AsRequired()
	}

	if pp.Tag.Deprecated {
		p.Description = "Deprecated"
	}

	kind := pp.Kind
	switch kind {
	case desc.Array:
		kind = pp.Element.Kind
	}

	switch kind {
	default:
		p.Typed("object", "")
	case desc.Array:
		p.Typed("array", "")
	case desc.Bool:
		p.Typed("boolean", "")
	case desc.Integer:
		p.Typed("integer", "")
	case desc.Float:
		p.Typed("number", "")
	case desc.Object:
		p.Typed("object", "")
	case desc.Map:
		p.Typed("object", "")
	case desc.String:
		p.Typed("string", "")
	}

	if len(pp.Tag.PossibleValues) > 0 {
		p.WithEnum(
			utils.Map(
				func(src string) any { return src },
				pp.Tag.PossibleValues,
			)...,
		)
	}

	return p
}

// fixPathForSwag converts the ronykit mux format urls to swagger url format.
// e.g. /some/path/:x1 --> /some/path/{x1}
func fixPathForSwag(path string) string {
	sb := strings.Builder{}
	for idx, p := range strings.Split(path, "/") {
		if idx > 0 {
			sb.WriteRune('/')
		}
		if strings.HasPrefix(p, ":") {
			sb.WriteRune('{')
			sb.WriteString(p[1:])
			sb.WriteRune('}')
		} else {
			sb.WriteString(p)
		}
	}

	return sb.String()
}

type schemaWrapper func(schema *spec.Schema) *spec.Schema

type schemaWrapperChain []schemaWrapper

func (chain *schemaWrapperChain) Add(w schemaWrapper) schemaWrapperChain {
	*chain = append(*chain, w)

	return *chain
}

func (chain schemaWrapperChain) Apply(schema *spec.Schema) spec.Schema {
	for _, w := range chain {
		schema = w(schema)
	}

	return *schema
}
