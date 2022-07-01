package stubgengo

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed tpl/dto.gotmpl
	tplFileDTO string
	tplDTO     *template.Template
)

func init() {
	tplDTO = template.Must(template.New("dto").Parse(tplFileDTO))
}
