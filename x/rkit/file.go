package rkit

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func GetExecDir() string {
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}

	return execDir
}

func GetExecName() string {
	return filepath.Base(os.Args[0])
}

func CopyFile(srcFile, dstFile string) error {
	const bufferSize = 1024 * 1024
	buf := make([]byte, bufferSize)

	return CopyFileWithBuffer(srcFile, dstFile, buf)
}

func CopyFileWithBuffer(srcFile, dstFile string, buf []byte) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer func(src *os.File) {
		_ = src.Close()
	}(src)
	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := dst.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}

func WriteYamlFile(filePath string, data any) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	enc := yaml.NewEncoder(f)
	err = enc.Encode(data)
	if err != nil {
		return err
	}

	return enc.Close()
}

func ReadYamlFile(filePath string, data any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	dec := yaml.NewDecoder(f)

	return dec.Decode(data)
}
