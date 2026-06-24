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
	// DestMapper optionally rewrites the destination for each entry. It
	// receives the entry path relative to SrcPathPrefix (forward slashes, no
	// leading separator; the root entry is the empty string) and returns the
	// destination path. Returning skip=true omits the entry. When DestMapper is
	// nil, the destination defaults to filepath.Join(DestPathPrefix, relPath).
	DestMapper func(relPath string) (dest string, skip bool)
}

func CopyDir(params CopyDirParams) error {
	return fs.WalkDir(
		params.FS, params.SrcPathPrefix,
		func(currPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relPath := strings.TrimPrefix(strings.TrimPrefix(currPath, params.SrcPathPrefix), "/")

			destPath, skip := params.resolveDest(relPath)
			if skip {
				return nil
			}

			if d.IsDir() {
				// Create a directory if it doesn't exist
				return os.MkdirAll(destPath, os.ModePerm)
			}

			if params.Callback != nil {
				params.Callback(currPath, d.IsDir())
			}

			// Ensure the parent directory exists. When DestMapper reroutes
			// entries (e.g. into backend/), the destination parent may not have
			// been created by a preceding directory walk entry.
			if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
				return err
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

func (p CopyDirParams) resolveDest(relPath string) (string, bool) {
	if p.DestMapper != nil {
		dest, skip := p.DestMapper(relPath)
		if skip {
			return "", true
		}

		return strings.TrimSuffix(dest, "tmpl"), false
	}

	return strings.TrimSuffix(filepath.Join(p.DestPathPrefix, relPath), "tmpl"), false
}

type CopyFileParams struct {
	FS             fs.FS
	SrcPath        string
	DestPath       string
	TemplateSuffix string
	TemplateInput  any
}

func CopyFile(p CopyFileParams) error {
	if p.TemplateInput != nil && strings.HasSuffix(p.SrcPath, p.TemplateSuffix) {
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
