package swagger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/go-openapi/spec"
	"github.com/rbretecher/go-postman-collection"
)

type Generator struct {
	s       *spec.Swagger
	tagName string
	title   string
	version string
	desc    string
}

func New(title, ver, desc string) *Generator {
	sg := &Generator{
		s:       &spec.Swagger{},
		title:   title,
		version: ver,
		desc:    desc,
	}
	sg.s.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Description: desc,
			Title:       title,
			Version:     ver,
		},
	}
	sg.s.Schemes = []string{"http", "https"}
	sg.s.Swagger = "2.0"

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
	for _, d := range descs {
		// extract the service description
		s := d.Desc()

		addSwaggerTag(sg.s, s)

		for _, c := range s.Contracts {
			c.PossibleErrors = append(c.PossibleErrors, s.PossibleErrors...)
			sg.addSwagOp(sg.s, s.Name, c)
		}
	}

	swaggerJSON, err := sg.s.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.Write(swaggerJSON)

	return err
}

func (sg *Generator) addSwagOp(swag *spec.Swagger, serviceName string, c desc.Contract) {
	if swag.Paths == nil {
		swag.Paths = &spec.Paths{
			Paths: map[string]spec.PathItem{},
		}
	}
	var contentType string
	switch c.Encoding {
	case kit.JSON:
		contentType = "application/json"
	case kit.Proto:
		contentType = "application/x-protobuf"
	case kit.MSG:
		contentType = "application/octet-stream"
	default:
		contentType = "application/json"
	}

	inType := reflect.Indirect(reflect.ValueOf(c.Input)).Type()
	outType := reflect.Indirect(reflect.ValueOf(c.Output)).Type()
	opID := c.Name
	op := spec.NewOperation(opID).
		WithTags(serviceName).
		WithProduces(contentType).
		WithConsumes(contentType).
		RespondsWith(
			http.StatusOK,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", outType.Name())),
				),
		)

	possibleErrors := map[int][]string{}
	for _, pe := range c.PossibleErrors {
		errType := reflect.Indirect(reflect.ValueOf(pe.Message)).Type()
		sg.addSwagDefinition(swag, errType)
		possibleErrors[pe.Code] = append(possibleErrors[pe.Code], pe.Item)
		op.RespondsWith(
			pe.Code,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", errType.Name())),
				).
				WithDescription(fmt.Sprintf("Items: %s", strings.Join(possibleErrors[pe.Code], ", "))),
		)
	}
	for _, sel := range c.RouteSelectors {
		restSel, ok := sel.Selector.(kit.RESTRouteSelector)
		if !ok {
			continue
		}

		sg.setSwagInput(op, restSel.GetPath(), inType)
		sg.addSwagDefinition(swag, inType)
		sg.addSwagDefinition(swag, outType)

		restPath := fixPathForSwag(restSel.GetPath())
		pathItem := swag.Paths.Paths[restPath]
		switch strings.ToUpper(restSel.GetMethod()) {
		case http.MethodGet:
			pathItem.Get = op

		case http.MethodDelete:
			pathItem.Delete = op
		case http.MethodPost:
			op.AddParam(
				spec.BodyParam(
					inType.Name(),
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", inType.Name())),
				),
			)
			pathItem.Post = op
		case http.MethodPut:
			op.AddParam(
				spec.BodyParam(
					inType.Name(),
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", inType.Name())),
				),
			)
			pathItem.Put = op
		case http.MethodPatch:
			op.AddParam(
				spec.BodyParam(
					inType.Name(),
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", inType.Name())),
				),
			)
			pathItem.Patch = op
		}
		swag.Paths.Paths[restPath] = pathItem
	}
}

func (sg *Generator) setSwagInput(op *spec.Operation, path string, inType reflect.Type) {
	if inType.Kind() == reflect.Ptr {
		inType = inType.Elem()
	}
	if inType.Kind() != reflect.Struct {
		return
	}

	var pathParams []string //nolint:prealloc
	for _, pp := range strings.Split(path, "/") {
		if !strings.HasPrefix(pp, ":") {
			continue
		}
		pathParam := strings.TrimPrefix(pp, ":")
		pathParams = append(pathParams, pathParam)
	}

	queue := []reflect.Type{inType}
	for j := 0; j < len(queue); j++ {
		inType := queue[j]
		for i := 0; i < inType.NumField(); i++ {
			if inType.Field(i).Anonymous {
				queue = append(queue, inType.Field(i).Type)

				continue
			}
			pt := getParsedStructTag(inType.Field(i).Tag, sg.tagName)
			if pt.Name == "" {
				continue
			}
			found := false
			for _, pathParam := range pathParams {
				if strings.ToLower(pt.Name) == strings.ToLower(pathParam) {
					found = true
				}
			}

			switch {
			case found:
				op.AddParam(
					setSwaggerParam(
						spec.PathParam(pt.Name),
						inType.Field(i).Type,
						pt.Optional,
					),
				)
			default:
				op.AddParam(
					setSwaggerParam(
						spec.QueryParam(pt.Name),
						inType.Field(i).Type,
						pt.Optional,
					),
				)
			}
		}
	}
}

func (sg *Generator) addSwagDefinition(swag *spec.Swagger, rType reflect.Type) {
	if rType.Kind() == reflect.Ptr {
		rType = rType.Elem()
	}
	if rType.Kind() != reflect.Struct {
		return
	}

	if swag.Definitions == nil {
		swag.Definitions = map[string]spec.Schema{}
	}

	def := spec.Schema{}
	def.Typed("object", "")

	queue := []reflect.Type{rType}
	for j := 0; j < len(queue); j++ {
		rType := queue[j]
		for i := 0; i < rType.NumField(); i++ {
			f := rType.Field(i)
			if f.Anonymous {
				queue = append(queue, f.Type)

				continue
			}
			fType := f.Type
			pt := getParsedStructTag(f.Tag, sg.tagName)
			if pt.Name == "" {
				continue
			}

			var wrapFuncChain schemaWrapperChain
			switch fType.Kind() {
			case reflect.Ptr:
				fType = fType.Elem()
				wrapFuncChain.Add(
					func(schema *spec.Schema) *spec.Schema {
						return schema
					},
				)
			case reflect.Slice:
				fType = fType.Elem()
				wrapFuncChain.Add(
					func(schema *spec.Schema) *spec.Schema {
						return spec.ArrayProperty(schema)
					},
				)
			default:
				wrapFuncChain.Add(
					func(schema *spec.Schema) *spec.Schema {
						return schema
					},
				)
			}

			if len(pt.PossibleValues) > 0 {
				wrapFuncChain.Add(
					func(schema *spec.Schema) *spec.Schema {
						for _, v := range pt.PossibleValues {
							schema.Enum = append(schema.Enum, v)
						}

						return schema
					},
				)
			}

		Switch:
			switch fType.Kind() {
			case reflect.String:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.StringProperty()))
			case reflect.Int8, reflect.Uint8:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.ArrayProperty(spec.Int8Property())))
			case reflect.Int32, reflect.Uint32:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.Int32Property()))
			case reflect.Int, reflect.Uint, reflect.Int64, reflect.Uint64:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.Int64Property()))
			case reflect.Float32:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.Float32Property()))
			case reflect.Float64:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.Float64Property()))
			case reflect.Struct:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.RefProperty(fmt.Sprintf("#/definitions/%s", fType.Name()))))
				sg.addSwagDefinition(swag, fType)
			case reflect.Bool:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.BoolProperty()))
			case reflect.Interface, reflect.Map:
				sub := &spec.Schema{}
				sub.Typed("object", "")
				def.SetProperty(pt.Name, wrapFuncChain.Apply(sub))
			case reflect.Ptr:
				fType = fType.Elem()

				goto Switch
			default:
				def.SetProperty(pt.Name, wrapFuncChain.Apply(spec.StringProperty()))
			}
		}

		swag.Definitions[rType.Name()] = def
	}
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
			sg.addPostmanItem(colItems, c)
		}
	}

	return col.Write(w, postman.V210)
}

func (sg *Generator) addPostmanItem(items *postman.Items, c desc.ParsedContract) {
	itm := postman.CreateItem(
		postman.Item{
			Name:                    c.Name,
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
	itm.Request = &postman.Request{
		URL: &postman.URL{
			Raw: fmt.Sprintf("{{baseURL}}%s", c.Path),
			Host: []string{
				"{{baseURL}}",
			},
			Path: strings.Split(c.Path, "/"),
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

func addSwaggerTag(swag *spec.Swagger, s *desc.Service) {
	swag.Tags = append(
		swag.Tags,
		spec.NewTag(s.Name, s.Description, nil),
	)
}

func setSwaggerParam(p *spec.Parameter, t reflect.Type, optional bool) *spec.Parameter {
	if optional {
		p.AsOptional()
	} else {
		p.AsRequired()
	}
	kind := t.Kind()
	switch kind {
	case reflect.Map:
		p.Typed("object", "")
	case reflect.Slice:
		switch t.Elem().Kind() {
		case reflect.String:
			p.Typed("string", kind.String())
		case reflect.Float64, reflect.Float32:
			p.Typed("number", kind.String())
		case reflect.Int8, reflect.Uint8:
			p.Typed("integer", "int8")
		case reflect.Int32, reflect.Uint32:
			p.Typed("integer", "int32")
		case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
			p.Typed("integer", "int64")
		default:
			return nil
		}
	case reflect.String:
		p.Typed("string", kind.String())
	case reflect.Float64, reflect.Float32:
		p.Typed("number", kind.String())
	case reflect.Int8, reflect.Uint8:
		p.Typed("integer", "int8")
	case reflect.Int32, reflect.Uint32:
		p.Typed("integer", "int32")
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		p.Typed("integer", "int64")
	case reflect.Bool:
		p.Typed("integer", "int64")
	default:
		p.Typed("boolean", "")
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
