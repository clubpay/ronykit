package swagger

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
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

		if ps.Origin.Name != "" {
			swag.Tags = append(
				swag.Tags,
				spec.NewTag(ps.Origin.Name, ps.Origin.Description, nil),
			)
		}

		for _, m := range ps.Messages() {
			addSwagDefinition(swag, ps.Origin.Name, m)
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

func (sg *Generator) SwaggerUI(svc ...desc.ServiceDesc) (fs.FS, error) {
	content := bytes.NewBuffer(nil)

	err := sg.WriteSwagTo(content, svc...)
	if err != nil {
		return nil, err
	}

	return &customFS{
		fs:          swaggerFS,
		folderName:  "swagger-ui",
		swaggerJSON: content.Bytes(),
		t:           time.Now(),
	}, nil
}

func (sg *Generator) ReDocUI(svc ...desc.ServiceDesc) (fs.FS, error) {
	content := bytes.NewBuffer(nil)

	err := sg.WriteSwagTo(content, svc...)
	if err != nil {
		return nil, err
	}

	return &customFS{
		fs:          redocFS,
		folderName:  "redoc-ui",
		swaggerJSON: content.Bytes(),
		t:           time.Now(),
	}, nil
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
	case kit.MultipartForm.Tag():
		contentType = "multipart/form-data"
	default:
		contentType = "application/json"
	}

	if c.Type != desc.REST {
		return
	}

	opID := definitionName(serviceName, c.SelectorName)

	op := spec.NewOperation(opID).
		WithProduces(contentType).
		WithConsumes(contentType)
	if serviceName != "" {
		op.Tags = []string{serviceName}
	}

	if c.Deprecated {
		op.Deprecate()
	}

	if c.OKResponse().Message.IsSpecial() {
		op.RespondsWith(
			http.StatusOK,
			spec.NewResponse(),
		)
	} else {
		op.RespondsWith(
			http.StatusOK,
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(serviceName, c.OKResponse().Message.Name))),
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
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(serviceName, r.Message.Name))),
				).
				WithDescription(fmt.Sprintf("Items: %s", strings.Join(possibleErrors[r.ErrCode], ", "))),
		)
	}

	if c.DefaultError != nil {
		op.WithDefaultResponse(
			spec.NewResponse().
				WithSchema(
					spec.RefProperty(
						fmt.Sprintf("#/definitions/%s", definitionName(serviceName, c.DefaultError.Message.Name)),
					),
				),
		)
	}

	switch c.Encoding {
	default:
		setSwagInput(op, c)
	case kit.MultipartForm.Tag():
		setSwagInputFormData(op, c)
	}

	restPath := fixPathForSwag(c.Path)
	pathItem := swag.Paths.Paths[restPath]

	switch strings.ToUpper(c.Method) {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodDelete:
		pathItem.Delete = op
	case http.MethodPost:
		if !c.Request.Message.IsSpecial() {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(serviceName, c.Request.Message.Name))),
				),
			)
		}

		pathItem.Post = op
	case http.MethodPut:
		if !c.Request.Message.IsSpecial() {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(serviceName, c.Request.Message.Name))),
				),
			)
		}

		pathItem.Put = op
	case http.MethodPatch:
		if !c.Request.Message.IsSpecial() {
			op.AddParam(
				spec.BodyParam(
					c.Request.Message.Name,
					spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(serviceName, c.Request.Message.Name))),
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

	var fields []desc.ParsedField

	for _, f := range c.Request.Message.Fields {
		if !f.Exported {
			continue
		}

		if f.Embedded {
			fields = append(fields, f.Element.Message.Fields...)
		} else {
			fields = append(fields, f)
		}
	}

	for _, field := range fields {
		if !field.Exported {
			continue
		}

		if c.IsPathParam(field.Name) {
			op.AddParam(
				setSwaggerParam(spec.PathParam(field.Name), field),
			)
		} else if c.Method == http.MethodGet || c.Method == http.MethodDelete {
			op.AddParam(
				setSwaggerParam(spec.QueryParam(field.Name), field),
			)
		}
	}

	for _, hdr := range c.Request.Headers {
		if hdr.Required {
			op.AddParam(
				spec.HeaderParam(hdr.Name).
					Typed("string", "").
					AsRequired(),
			)
		} else {
			op.AddParam(
				spec.HeaderParam(hdr.Name).
					Typed("string", "").
					AsOptional(),
			)
		}
	}
}

func setSwagInputFormData(op *spec.Operation, c desc.ParsedContract) {
	for _, m := range c.Request.Message.Meta.Fields {
		if m.FormData == nil {
			continue
		}

		op.AddParam(
			spec.FormDataParam(m.FormData.Name).Typed(m.FormData.Type, ""),
		)
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

func addSwagDefinition(swag *spec.Swagger, svcName string, m desc.ParsedMessage) {
	if swag.Definitions == nil {
		swag.Definitions = map[string]spec.Schema{}
	}

	swag.Definitions[definitionName(svcName, m.Name)] = toSwagDefinition(svcName, m)
}

func toSwagDefinition(svcName string, m desc.ParsedMessage) spec.Schema {
	def := spec.Schema{}
	def.Typed("object", "")

	fields := list.New()
	for _, f := range m.Fields {
		fields.PushFront(f)
	}

	idx := 0

	for fields.Back() != nil {
		p := fields.Remove(fields.Back()).(desc.ParsedField) //nolint:errcheck,forcetypeassert

		// handle []byte as a special case of string[base64]
		if p.Element.Kind == desc.Array && p.Element.Element.Kind == desc.Byte {
			setProperty(&def, p.Name, *spec.StrFmtProperty("base64"), idx)
			idx++

			continue
		}

		name, kind, wrapFuncChain := getWrapFunc(p, m.Meta.Fields[p.GoName])

		switch kind {
		default:
			setProperty(&def, name, wrapFuncChain.Apply(spec.StringProperty()), idx)
		case desc.Object:
			if p.Embedded {
				for _, f := range p.Element.Message.Fields {
					fields.PushBack(f)
				}
			} else {
				setProperty(
					&def, p.Name,
					wrapFuncChain.Apply(spec.RefProperty(fmt.Sprintf("#/definitions/%s", definitionName(svcName, name)))),
					idx,
				)
			}
		case desc.String:
			setProperty(&def, p.Name, wrapFuncChain.Apply(spec.StringProperty()), idx)
		case desc.Float:
			setProperty(&def, p.Name, wrapFuncChain.Apply(spec.Float64Property()), idx)
		case desc.Integer:
			setProperty(&def, p.Name, wrapFuncChain.Apply(spec.Int64Property()), idx)
		case desc.Bool:
			setProperty(&def, p.Name, wrapFuncChain.Apply(spec.BoolProperty()), idx)
		case desc.Byte:
			setProperty(&def, p.Name, wrapFuncChain.Apply(spec.Int8Property()), idx)
		}

		idx++
	}

	return def
}

func setProperty(def *spec.Schema, name string, prop spec.Schema, idx int) {
	prop.AddExtension("x-order", float64(idx))
	def.SetProperty(name, prop)
}

func getWrapFunc(p desc.ParsedField, meta desc.FieldMeta) (string, desc.Kind, schemaWrapperChain) {
	var (
		wrapFuncChain = schemaWrapperChain{}
		name          string
	)

	msg := p.Element.Message
	kind := p.Element.Kind
	elem := p.Element

Loop:
	switch kind {
	default:
	case desc.Object:
		name = msg.Name
	case desc.Map:
		wrapFuncChain = wrapFuncChain.Add(spec.MapProperty)
		kind = elem.Element.Kind
		msg = elem.Element.Message
		elem = elem.Element

		goto Loop
	case desc.Array:
		wrapFuncChain = wrapFuncChain.Add(spec.ArrayProperty)
		kind = elem.Element.Kind
		msg = elem.Element.Message
		elem = elem.Element

		goto Loop
	}

	possibleValues := p.Tag.PossibleValues
	if meta.Enum != nil {
		possibleValues = meta.Enum
	}

	enum := utils.Map(
		func(v string) any { return v },
		possibleValues,
	)
	if len(enum) > 0 {
		wrapFuncChain = wrapFuncChain.Add(
			func(schema *spec.Schema) *spec.Schema {
				schema.Enum = enum

				return schema
			},
		)
	}

	if p.Tag.Optional || p.Optional || meta.Optional {
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

	if p.Tag.Deprecated || meta.Deprecated {
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

	if c.Deprecated {
		itm.Description = "[Deprecated] " + itm.Description
	}

	for _, pp := range c.PathParams {
		v := &postman.Variable{
			Name: pp,
			Key:  pp,
		}
		for _, p := range c.Request.Message.Fields {
			if p.Name == pp {
				v.Type = string(p.Element.Kind)
				v.Value = p.SampleValue

				break
			}
		}

		itm.Variables = append(itm.Variables, v)
	}

	var queryParams []*postman.QueryParam

	if len(c.PathParams) > len(c.Request.Message.Fields) || c.Method == http.MethodGet {
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
	if pp.Tag.Optional || pp.Optional {
		p.AsOptional()
	} else {
		p.AsRequired()
	}

	if pp.Tag.Deprecated {
		p.Description = "Deprecated"
	}

	kind := pp.Element.Kind
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
// for example, /some/path/:x1 --> /some/path/{x1}
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

func definitionName(serviceName, name string) string {
	if serviceName == "" {
		return name
	}

	return serviceName + "." + name
}
