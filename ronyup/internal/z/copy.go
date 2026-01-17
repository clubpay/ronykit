package z

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type CopyDirParams struct {
	FS             fs.FS
	SrcPathPrefix  string
	DestPathPrefix string
	TemplateInput  any
	Callback       func(filePath string, dir bool)
}

func CopyDir(params CopyDirParams) error {
	return fs.WalkDir(
		params.FS, params.SrcPathPrefix,
		func(currPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			srcPath := strings.TrimPrefix(currPath, params.SrcPathPrefix)
			destPath := strings.TrimSuffix(filepath.Join(params.DestPathPrefix, srcPath), "tmpl")

			if d.IsDir() {
				// Create a directory if it doesn't exist
				return os.MkdirAll(destPath, os.ModePerm)
			}

			if params.Callback != nil {
				params.Callback(currPath, d.IsDir())
			}

			return CopyFile(CopyFileParams{
				FS:             params.FS,
				SrcPath:        currPath,
				DestPath:       destPath,
				TemplateSuffix: "tmpl",
				TemplateInput:  params.TemplateInput,
			})
		})
}

type CopyFileParams struct {
	FS             fs.FS
	SrcPath        string
	DestPath       string
	TemplateSuffix string
	TemplateInput  any
}

func CopyFile(p CopyFileParams) error {
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
