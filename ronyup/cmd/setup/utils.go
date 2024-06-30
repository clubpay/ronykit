package setup

import (
	"io"
	"os"
)

func isEmptyDir(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}

	return false
}
