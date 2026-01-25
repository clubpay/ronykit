package template

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GoPackage struct {
	Name      string
	Dir       string
	Files     []string
	Types     []GoType
	Functions []GoFunc
	Comments  []GoComment
}

type GoTypeKind string

const (
	GoTypeStruct    GoTypeKind = "struct"
	GoTypeInterface GoTypeKind = "interface"
	GoTypeAlias     GoTypeKind = "alias"
	GoTypeOther     GoTypeKind = "other"
)

type GoType struct {
	Name     string
	Kind     GoTypeKind
	Exported bool
	File     string
	Doc      string
	Comments []GoComment

	Underlying string
	Fields     []GoField
	Methods    []GoFunc
}

type GoFunc struct {
	Name         string
	Exported     bool
	File         string
	Doc          string
	Comments     []GoComment
	Receiver     string
	ReceiverType string
	Params       []GoParam
	Results      []GoParam
}

type GoField struct {
	Name     string
	Type     string
	Tag      string
	Embedded bool
	Exported bool
	Doc      string
	Comments []GoComment
}

type GoParam struct {
	Name string
	Type string
}

type GoComment struct {
	Text string
	Kind string
	Line int
}

type CommentTag struct {
	Name  string
	Value string
	Line  int
}

// ParseGoPath collects package metadata for a module path or a single Go file.
// Example: pkgs, err := ParseGoPath(".")
func ParseGoPath(root string) ([]GoPackage, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return parseGoFile(root)
	}

	return parseGoPackages(root)
}

// TypesByName returns a map keyed by type name for quick lookups.
func (p GoPackage) TypesByName() map[string]GoType {
	out := make(map[string]GoType, len(p.Types))
	for _, t := range p.Types {
		if t.Name == "" {
			continue
		}
		out[t.Name] = t
	}
	return out
}

// FindType looks up a type by name in the package.
func (p GoPackage) FindType(name string) (GoType, bool) {
	if name == "" {
		return GoType{}, false
	}
	for _, t := range p.Types {
		if t.Name == name {
			return t, true
		}
	}
	return GoType{}, false
}

// MethodsOf returns all methods whose receiver matches the given type name.
// Example: methods := pkg.MethodsOf("User")
func (p GoPackage) MethodsOf(typeName string) []GoFunc {
	if typeName == "" {
		return nil
	}
	want := typeName
	wantPtr := "*" + typeName
	var out []GoFunc
	for _, fn := range p.Functions {
		if fn.ReceiverType == want || fn.ReceiverType == wantPtr {
			out = append(out, fn)
		}
	}
	return out
}

// FuncsByReceiver groups methods by their receiver type.
// Example: receivers := pkg.FuncsByReceiver(); receivers["*User"]
func (p GoPackage) FuncsByReceiver() map[string][]GoFunc {
	out := make(map[string][]GoFunc)
	for _, fn := range p.Functions {
		if fn.ReceiverType == "" {
			continue
		}
		out[fn.ReceiverType] = append(out[fn.ReceiverType], fn)
	}
	return out
}

// FieldsOf returns the fields for a named type, if present.
// Example: fields := pkg.FieldsOf("User")
func (p GoPackage) FieldsOf(typeName string) []GoField {
	t, ok := p.FindType(typeName)
	if !ok {
		return nil
	}
	return t.Fields
}

// parseGoFile parses a single Go file into a minimal GoPackage.
func parseGoFile(path string) ([]GoPackage, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pkg := GoPackage{
		Name:  file.Name.Name,
		Dir:   filepath.Dir(path),
		Files: []string{path},
	}
	collectFromFile(&pkg, fset, file)
	return []GoPackage{pkg}, nil
}

// parseGoPackages loads all non-test packages under root using go/packages.
func parseGoPackages(root string) ([]GoPackage, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedModule,
		Dir:   root,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}
	if err := firstPackageError(pkgs); err != nil {
		return nil, err
	}

	var out []GoPackage
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 {
			continue
		}
		pkgInfo := GoPackage{
			Name:  pkg.Name,
			Dir:   packageDir(pkg),
			Files: append([]string{}, pkg.GoFiles...),
		}
		for _, file := range pkg.Syntax {
			collectFromFile(&pkgInfo, pkg.Fset, file)
		}
		pkgInfo.Types = dedupeTypes(pkgInfo.Types)
		pkgInfo.Functions = dedupeFuncs(pkgInfo.Functions)
		out = append(out, pkgInfo)
	}

	return out, nil
}

// packageDir derives a stable directory path from the first available file.
func packageDir(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	if len(pkg.CompiledGoFiles) > 0 {
		return filepath.Dir(pkg.CompiledGoFiles[0])
	}
	return ""
}

// firstPackageError returns the first load error across all packages.
func firstPackageError(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			return fmt.Errorf("packages.Load: %w", pkgErr)
		}
	}
	return nil
}

func collectFromFile(pkg *GoPackage, fset *token.FileSet, file *ast.File) {
	pkg.Comments = append(pkg.Comments, parseCommentGroups(fset, file.Comments)...)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				kind, fields, methods := parseTypeSpec(fset, ts)
				pkg.Types = append(pkg.Types, GoType{
					Name:       ts.Name.Name,
					Kind:       kind,
					Exported:   ast.IsExported(ts.Name.Name),
					File:       fset.Position(ts.Pos()).Filename,
					Doc:        commentGroupText(ts.Doc),
					Comments:   parseCommentGroup(fset, ts.Comment),
					Underlying: exprString(fset, ts.Type),
					Fields:     fields,
					Methods:    methods,
				})
			}
		case *ast.FuncDecl:
			fn := GoFunc{
				Name:     d.Name.Name,
				Exported: ast.IsExported(d.Name.Name),
				File:     fset.Position(d.Pos()).Filename,
				Doc:      commentGroupText(d.Doc),
				Comments: parseCommentGroup(fset, d.Doc),
				Params:   fieldListToParams(fset, d.Type.Params),
				Results:  fieldListToParams(fset, d.Type.Results),
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := d.Recv.List[0]
				if len(recv.Names) > 0 {
					fn.Receiver = recv.Names[0].Name
				}
				fn.ReceiverType = exprString(fset, recv.Type)
			}
			pkg.Functions = append(pkg.Functions, fn)
		}
	}
}

func parseTypeSpec(fset *token.FileSet, ts *ast.TypeSpec) (GoTypeKind, []GoField, []GoFunc) {
	switch t := ts.Type.(type) {
	case *ast.StructType:
		return GoTypeStruct, parseStructFields(fset, t.Fields), nil
	case *ast.InterfaceType:
		fields, methods := parseInterfaceFields(fset, t.Methods)
		return GoTypeInterface, fields, methods
	default:
		if ts.Assign.IsValid() {
			return GoTypeAlias, nil, nil
		}
		return GoTypeOther, nil, nil
	}
}

func parseStructFields(fset *token.FileSet, fl *ast.FieldList) []GoField {
	if fl == nil {
		return nil
	}

	var out []GoField
	for _, field := range fl.List {
		typ := exprString(fset, field.Type)
		tag := ""
		if field.Tag != nil {
			tag = strings.Trim(field.Tag.Value, "`")
		}
		doc := commentGroupText(field.Doc)
		comments := parseCommentGroup(fset, field.Comment)

		if len(field.Names) == 0 {
			out = append(out, GoField{
				Type:     typ,
				Tag:      tag,
				Embedded: true,
				Exported: ast.IsExported(typeNameFromExpr(field.Type)),
				Doc:      doc,
				Comments: comments,
			})
			continue
		}

		for _, name := range field.Names {
			out = append(out, GoField{
				Name:     name.Name,
				Type:     typ,
				Tag:      tag,
				Exported: name.IsExported(),
				Doc:      doc,
				Comments: comments,
			})
		}
	}

	return out
}

func parseInterfaceFields(fset *token.FileSet, fl *ast.FieldList) ([]GoField, []GoFunc) {
	if fl == nil {
		return nil, nil
	}

	var fields []GoField
	var methods []GoFunc
	for _, field := range fl.List {
		switch ft := field.Type.(type) {
		case *ast.FuncType:
			name := ""
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			methods = append(methods, GoFunc{
				Name:     name,
				Exported: ast.IsExported(name),
				Doc:      commentGroupText(field.Doc),
				Comments: parseCommentGroup(fset, field.Comment),
				Params:   fieldListToParams(fset, ft.Params),
				Results:  fieldListToParams(fset, ft.Results),
			})
		default:
			typ := exprString(fset, field.Type)
			fields = append(fields, GoField{
				Type:     typ,
				Embedded: len(field.Names) == 0,
				Exported: ast.IsExported(typeNameFromExpr(field.Type)),
				Doc:      commentGroupText(field.Doc),
				Comments: parseCommentGroup(fset, field.Comment),
			})
		}
	}

	return fields, methods
}

func fieldListToParams(fset *token.FileSet, fl *ast.FieldList) []GoParam {
	if fl == nil {
		return nil
	}

	var out []GoParam
	for _, field := range fl.List {
		typ := exprString(fset, field.Type)
		if len(field.Names) == 0 {
			out = append(out, GoParam{Type: typ})
			continue
		}
		for _, name := range field.Names {
			out = append(out, GoParam{Name: name.Name, Type: typ})
		}
	}

	return out
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, expr); err != nil {
		return fmt.Sprintf("%T", expr)
	}
	return buf.String()
}

func typeNameFromExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return typeNameFromExpr(t.X)
	default:
		return ""
	}
}

func commentGroupText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return strings.TrimSpace(cg.Text())
}

func parseCommentGroups(fset *token.FileSet, groups []*ast.CommentGroup) []GoComment {
	if len(groups) == 0 {
		return nil
	}

	var out []GoComment
	for _, group := range groups {
		out = append(out, parseCommentGroup(fset, group)...)
	}

	return out
}

func parseCommentGroup(fset *token.FileSet, group *ast.CommentGroup) []GoComment {
	if group == nil {
		return nil
	}

	var out []GoComment
	for _, c := range group.List {
		pos := fset.Position(c.Pos())
		out = append(out, GoComment{
			Text: strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), "/*")),
			Kind: commentKind(c),
			Line: pos.Line,
		})
	}

	return out
}

func commentKind(c *ast.Comment) string {
	if strings.HasPrefix(c.Text, "//") {
		return "line"
	}
	return "block"
}

func (c GoComment) Tags() []CommentTag {
	return parseCommentTags(c.Text, c.Line)
}

// HasTag reports whether the comment contains a tag like "@name" or "@name: value".
func (c GoComment) HasTag(name string) bool {
	_, ok := c.TagValue(name)
	return ok
}

// TagValue returns the value for a tag in the comment.
// Example: "// @owner: platform" -> TagValue("owner") == "platform".
func (c GoComment) TagValue(name string) (string, bool) {
	tags := parseCommentTags(c.Text, c.Line)
	for _, tag := range tags {
		if strings.EqualFold(tag.Name, name) {
			return tag.Value, true
		}
	}
	return "", false
}

func (t GoType) CommentTags() []CommentTag {
	return parseCommentTagsFromDocAndInline(t.Doc, t.Comments)
}

// HasCommentTag reports whether the type's doc or inline comments include a tag.
// Example:
//
//	// User is a domain model.
//	// @domain: identity
//	type User struct {}
func (t GoType) HasCommentTag(name string) bool {
	_, ok := t.CommentTagValue(name)
	return ok
}

// CommentTagValue returns the value for a tag from the type's doc or inline comments.
func (t GoType) CommentTagValue(name string) (string, bool) {
	return findCommentTagValue(parseCommentTagsFromDocAndInline(t.Doc, t.Comments), name)
}

// MethodsFrom filters package-level functions to those with receivers matching this type.
// Example:
//
//	for _, m := range myType.MethodsFrom(pkg.Functions) { fmt.Println(m.Name) }
func (t GoType) MethodsFrom(funcs []GoFunc) []GoFunc {
	if t.Name == "" {
		return nil
	}
	want := t.Name
	wantPtr := "*" + t.Name
	var out []GoFunc
	for _, fn := range funcs {
		if fn.ReceiverType == want || fn.ReceiverType == wantPtr {
			out = append(out, fn)
		}
	}
	return out
}

func (f GoFunc) CommentTags() []CommentTag {
	return parseCommentTagsFromDocAndInline(f.Doc, f.Comments)
}

// HasCommentTag reports whether the function's doc or inline comments include a tag.
// Example:
//
//	// CreateUser creates a new user.
//	// @role: admin
//	func CreateUser(...) {}
func (f GoFunc) HasCommentTag(name string) bool {
	_, ok := f.CommentTagValue(name)
	return ok
}

// CommentTagValue returns the value for a tag from the function's doc or inline comments.
func (f GoFunc) CommentTagValue(name string) (string, bool) {
	return findCommentTagValue(parseCommentTagsFromDocAndInline(f.Doc, f.Comments), name)
}

func (f GoField) CommentTags() []CommentTag {
	return parseCommentTagsFromDocAndInline(f.Doc, f.Comments)
}

// HasCommentTag reports whether the field's doc or inline comments include a tag.
// Example:
//
//	// @validate: required
//	Name string
func (f GoField) HasCommentTag(name string) bool {
	_, ok := f.CommentTagValue(name)
	return ok
}

// CommentTagValue returns the value for a tag from the field's doc or inline comments.
func (f GoField) CommentTagValue(name string) (string, bool) {
	return findCommentTagValue(parseCommentTagsFromDocAndInline(f.Doc, f.Comments), name)
}

// TagValue returns the raw struct tag value for name (e.g. "json", "db").
func (f GoField) TagValue(name string) (string, bool) {
	if f.Tag == "" {
		return "", false
	}
	value, ok := reflect.StructTag(f.Tag).Lookup(name)
	return value, ok
}

// TagName returns the first segment of a struct tag value.
// Example: `json:"name,omitempty"` -> TagName("json") == "name".
func (f GoField) TagName(name string) (string, bool) {
	value, ok := f.TagValue(name)
	if !ok || value == "" {
		return "", ok
	}
	parts := strings.Split(value, ",")
	return parts[0], true
}

func dedupeTypes(types []GoType) []GoType {
	seen := make(map[string]struct{}, len(types))
	var out []GoType
	for _, t := range types {
		if _, ok := seen[t.Name]; ok {
			continue
		}
		seen[t.Name] = struct{}{}
		out = append(out, t)
	}
	return out
}

func dedupeFuncs(funcs []GoFunc) []GoFunc {
	seen := make(map[string]struct{}, len(funcs))
	var out []GoFunc
	for _, fn := range funcs {
		key := fn.Name
		if fn.ReceiverType != "" {
			key = fn.ReceiverType + "." + fn.Name
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, fn)
	}
	return out
}

func parseCommentTagsFromDocAndInline(doc string, comments []GoComment) []CommentTag {
	var out []CommentTag
	out = append(out, parseCommentTagsFromDoc(doc)...)
	for _, c := range comments {
		out = append(out, c.Tags()...)
	}
	return out
}

func parseCommentTagsFromDoc(doc string) []CommentTag {
	if strings.TrimSpace(doc) == "" {
		return nil
	}
	lines := strings.Split(doc, "\n")
	var out []CommentTag
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, parseCommentTags(line, 0)...)
	}
	return out
}

func parseCommentTags(text string, line int) []CommentTag {
	if !strings.Contains(text, "@") {
		return nil
	}
	var out []CommentTag
	for i := 0; i < len(text); {
		at := strings.IndexByte(text[i:], '@')
		if at == -1 {
			break
		}
		at += i
		if at > 0 && !isSpace(text[at-1]) {
			i = at + 1
			continue
		}
		nameStart := at + 1
		if nameStart >= len(text) || !isTagChar(text[nameStart]) {
			i = at + 1
			continue
		}
		nameEnd := nameStart
		for nameEnd < len(text) && isTagChar(text[nameEnd]) {
			nameEnd++
		}
		name := text[nameStart:nameEnd]
		valueStart := nameEnd
		for valueStart < len(text) && isSpace(text[valueStart]) {
			valueStart++
		}
		if valueStart < len(text) && text[valueStart] == ':' {
			valueStart++
			for valueStart < len(text) && isSpace(text[valueStart]) {
				valueStart++
			}
		}
		valueEnd := nextTagStart(text, valueStart)
		value := strings.TrimSpace(text[valueStart:valueEnd])
		if name == "" {
			i = at + 1
			continue
		}
		out = append(out, CommentTag{
			Name:  name,
			Value: value,
			Line:  line,
		})
		if valueEnd <= at {
			i = at + 1
			continue
		}
		i = valueEnd
	}
	return out
}

func nextTagStart(text string, start int) int {
	for i := start; i < len(text); i++ {
		if text[i] != '@' {
			continue
		}
		if i > 0 && !isSpace(text[i-1]) {
			continue
		}
		if i+1 < len(text) && isTagChar(text[i+1]) {
			return i
		}
	}
	return len(text)
}

func isTagChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-' || b == '_'
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func findCommentTagValue(tags []CommentTag, name string) (string, bool) {
	for _, tag := range tags {
		if strings.EqualFold(tag.Name, name) {
			return tag.Value, true
		}
	}
	return "", false
}
