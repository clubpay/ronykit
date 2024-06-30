package swagger

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

//go:embed internal/swagger-ui
var swaggerFS embed.FS

var _ fs.FS = (*customFS)(nil)

type customFS struct {
	fs          embed.FS
	swaggerJSON []byte
	t           time.Time
}

func (c customFS) Open(name string) (fs.File, error) {
	if name == "swagger.json" {
		return &swaggerFile{
			buf: bytes.NewBuffer(c.swaggerJSON),
			t:   c.t,
		}, nil
	}

	return c.fs.Open(filepath.Join("internal/swagger-ui", name))
}

var (
	_ fs.File     = (*swaggerFile)(nil)
	_ fs.FileInfo = (*swaggerFile)(nil)
)

type swaggerFile struct {
	buf *bytes.Buffer
	t   time.Time
}

func (s swaggerFile) Name() string {
	return "swagger.json"
}

func (s swaggerFile) Size() int64 {
	return int64(s.buf.Len())
}

func (s swaggerFile) Mode() fs.FileMode {
	return os.ModePerm
}

func (s swaggerFile) ModTime() time.Time {
	return s.t
}

func (s swaggerFile) IsDir() bool {
	return false
}

func (s swaggerFile) Sys() any {
	return nil
}

func (s swaggerFile) Stat() (fs.FileInfo, error) {
	return s, nil
}

func (s *swaggerFile) Read(b []byte) (int, error) {
	return s.buf.Read(b)
}

func (s swaggerFile) Close() error {
	return nil
}
