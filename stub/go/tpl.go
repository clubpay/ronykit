package stubgengo

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed tpl/stub.gotmpl
	tplFileStub string
	tplStub     *template.Template
)

func init() {
	tplStub = template.Must(template.New("stub").Parse(tplFileStub))
}
