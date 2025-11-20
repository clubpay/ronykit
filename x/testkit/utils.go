package testkit

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/clubpay/ronykit/x/rkit"
)

func FolderContent(path string) []string {
	_, srcCodePath, _, _ := runtime.Caller(1) //nolint:dogsled

	var files []string

	_ = filepath.WalkDir(
		filepath.Join(filepath.Dir(srcCodePath), path),
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			files = append(files, rkit.B2S(rkit.Must(os.ReadFile(path))))

			return nil
		},
	)

	return files
}
