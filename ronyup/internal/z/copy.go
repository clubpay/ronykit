package z

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type CopyParams struct {
	FS             fs.FS
	SrcPath        string
	DestPath       string
	TemplateSuffix string
	TemplateInput  any
}

func Copy(p CopyParams) error {
	if p.TemplateInput != nil {
		tmplBytes, err := fs.ReadFile(p.FS, p.SrcPath)
		if err != nil {
			return err
		}

		tmpl, err := template.New(p.SrcPath).Funcs(sprig.FuncMap()).Parse(string(tmplBytes))
		if err != nil {
			return err
		}

		destFile, err := os.Create(p.DestPath)
		if err != nil {
			return err
		}
		defer func(destFile *os.File) {
			_ = destFile.Close()
		}(destFile)

		// Provide data to execute the template here if necessary

		return tmpl.Execute(destFile, p.TemplateInput)
	}

	// Otherwise, copy the file
	srcFile, err := p.FS.Open(p.SrcPath)
	if err != nil {
		return err
	}
	defer func(srcFile fs.File) {
		_ = srcFile.Close()
	}(srcFile)

	destFile, err := os.Create(p.DestPath)
	if err != nil {
		return err
	}
	defer func(destFile *os.File) {
		_ = destFile.Close()
	}(destFile)

	_, err = io.Copy(destFile, srcFile)

	return err
}

func IsEmptyDir(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = f.Readdirnames(1)

	return errors.Is(err, io.EOF)
}
