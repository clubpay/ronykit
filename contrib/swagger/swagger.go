package swagger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/go-openapi/spec"
	"github.com/rbretecher/go-postman-collection"
)

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

		for _, c := range ps.Contracts {
			sg.addSwagOp(swag, ps.Origin.Name, c)
		}

		for _, m := range ps.Messages() {
			sg.addSwagDefinition(swag, m)
		}
	}

	swaggerJSON, err := swag.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.Write(swaggerJSON)

	return err
}

func (sg *Generator) addSwagOp(swag *spec.Swagger, serviceName string, c desc.ParsedContract) {
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
		WithConsumes(contentType).
		RespondsWith(
			http.StatusOK,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.OKResponse().Message.Name)),
				),
		)

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

	sg.setSwagInput(op, c)

	restPath := fixPathForSwag(c.Path)
	pathItem := swag.Paths.Paths[restPath]
	switch strings.ToUpper(c.Method) {
	case http.MethodGet:
		pathItem.Get = op

	case http.MethodDelete:
		pathItem.Delete = op
	case http.MethodPost:
		op.AddParam(
			spec.BodyParam(
				c.Request.Message.Name,
				spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
			),
		)
		pathItem.Post = op
	case http.MethodPut:
		op.AddParam(
			spec.BodyParam(
				c.Request.Message.Name,
				spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
			),
		)
		pathItem.Put = op
	case http.MethodPatch:
		op.AddParam(
			spec.BodyParam(
				c.Request.Message.Name,
				spec.RefProperty(fmt.Sprintf("#/definitions/%s", c.Request.Message.Name)),
			),
		)
		pathItem.Patch = op
	}
	swag.Paths.Paths[restPath] = pathItem
}

func (sg *Generator) setSwagInput(op *spec.Operation, c desc.ParsedContract) {
	if len(c.Request.Message.Params) == 0 {
		return
	}

	for _, p := range c.Request.Message.Params {
		if c.IsPathParam(p.Name) {
			op.AddParam(
				setSwaggerParam(spec.PathParam(p.Name), p),
			)
		} else if c.Method == http.MethodGet {
			op.AddParam(
				setSwaggerParam(spec.QueryParam(p.Name), p),
			)
		}
	}
}

func (sg *Generator) addSwagDefinition(swag *spec.Swagger, m desc.ParsedMessage) {
	if swag.Definitions == nil {
		swag.Definitions = map[string]spec.Schema{}
	}

	def := spec.Schema{}
	def.Typed("object", "")
	for _, p := range m.Params {
		var wrapFuncChain schemaWrapperChain

		kind := p.Kind
		keepGoing := true
		for keepGoing {
			switch kind {
			case desc.Map:
				wrapFuncChain.Add(spec.MapProperty)
				kind = p.SubKind
			case desc.Array:
				wrapFuncChain.Add(spec.ArrayProperty)
				kind = p.SubKind
			default:
				keepGoing = false
			}
		}
		switch kind {
		case desc.Object:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.RefProperty(fmt.Sprintf("#/definitions/%s", p.Message.Name))))
		case desc.String:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.StringProperty()))
		case desc.Float:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.Float64Property()))
		case desc.Integer:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.Int64Property()))
		default:
			def.SetProperty(p.Name, wrapFuncChain.Apply(spec.StringProperty()))
		}
	}

	swag.Definitions[m.Name] = def
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
				sg.addPostmanItem(colItems, c)
			}
		}
	}

	return col.Write(w, postman.V210)
}

func (sg *Generator) addPostmanItem(items *postman.Items, c desc.ParsedContract) {
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
		for _, p := range c.Request.Message.Params {
			if p.Name == pp {
				v.Type = string(p.Kind)
				v.Value = p.SampleValue

				break
			}
		}
		itm.Variables = append(itm.Variables, v)
	}

	var queryParams []*postman.QueryParam
	if len(c.PathParams) > len(c.Request.Message.Params) && c.Method == "GET" {
		for _, p := range c.Request.Message.Params {
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

	items.AddItem(itm)
}

func setSwaggerParam(p *spec.Parameter, pp desc.ParsedParam) *spec.Parameter {
	if pp.Tag.Optional {
		p.AsOptional()
	} else {
		p.AsRequired()
	}

	switch pp.Kind {
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
