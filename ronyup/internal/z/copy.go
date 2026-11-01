package z

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/clubpay/ronykit/x/rkit"

	"github.com/Masterminds/sprig/v3"
)

type CopyDirParams struct {
	FS             fs.FS
	SrcPathPrefix  string
	DestPathPrefix string
	// DestRoot is the trusted boundary for destination writes. It defaults to
	// DestPathPrefix; set it when copying into a nested directory.
	DestRoot      string
	TemplateInput any
	// Callback is called with the destination path of each file written.
	Callback func(filePath string, dir bool)
	// SkipExisting skips copying when the destination file already exists.
	SkipExisting bool
	// DestMapper optionally rewrites the destination for each entry. It
	// receives the entry path relative to SrcPathPrefix (forward slashes, no
	// leading separator; the root entry is the empty string) and returns the
	// destination path. Returning skip=true omits the entry. When DestMapper is
	// nil, the destination defaults to filepath.Join(DestPathPrefix, relPath).
	DestMapper func(relPath string) (dest string, skip bool)
}

func CopyDir(params CopyDirParams) error {
	destRoot := params.DestRoot
	destRoot = rkit.Coalesce(destRoot, params.DestPathPrefix)

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

			if err := CheckPath(destRoot, destPath); err != nil {
				return err
			}

			if d.IsDir() {
				return os.MkdirAll(destPath, 0o755)
			}

			if params.SkipExisting {
				if _, err := os.Lstat(destPath); err == nil {
					return nil
				} else if !os.IsNotExist(err) {
					return err
				}
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}

			if err := CopyFile(CopyFileParams{
				FS:             params.FS,
				SrcPath:        currPath,
				DestPath:       destPath,
				DestRoot:       destRoot,
				TemplateSuffix: "tmpl",
				TemplateInput:  params.TemplateInput,
			}); err != nil {
				return err
			}

			if params.Callback != nil {
				params.Callback(destPath, false)
			}

			return nil
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
	DestRoot       string
	TemplateSuffix string
	TemplateInput  any
}

func CopyFile(p CopyFileParams) error {
	data, err := fs.ReadFile(p.FS, p.SrcPath)
	if err != nil {
		return err
	}

	if p.TemplateInput != nil && strings.HasSuffix(p.SrcPath, p.TemplateSuffix) {
		tmpl, err := template.New(p.SrcPath).Funcs(sprig.FuncMap()).Parse(string(data))
		if err != nil {
			return err
		}

		var out bytes.Buffer
		if err := tmpl.Execute(&out, p.TemplateInput); err != nil {
			return err
		}

		data = out.Bytes()
	}

	root := p.DestRoot
	if root == "" {
		root = filepath.Dir(p.DestPath)
	}

	mode := fs.FileMode(0o644)
	if info, err := fs.Stat(p.FS, p.SrcPath); err == nil && info.Mode().Perm()&0o111 != 0 {
		mode = 0o755
	}

	if strings.HasSuffix(p.DestPath, ".sh") {
		mode = 0o755
	}

	if info, err := os.Stat(p.DestPath); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	return WriteFile(root, p.DestPath, data, mode)
}

// WriteFile atomically writes a file below root without following symlinks in
// existing destination components.
func WriteFile(root, path string, data []byte, mode fs.FileMode) error {
	if err := CheckPath(root, path); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return writeFileAtomic(path, data, mode)
}

func writeFileAtomic(path string, data []byte, mode fs.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".ronyup-*")
	if err != nil {
		return err
	}

	tmp := f.Name()
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}()

	if err := f.Chmod(mode); err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, path)
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
