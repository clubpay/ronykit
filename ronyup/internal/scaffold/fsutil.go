package scaffold

import "os"

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

// FileExists reports whether path exists on disk.
func FileExists(path string) bool { return fileExists(path) }

// IsDir reports whether path exists and is a directory.
func IsDir(path string) bool { return isDir(path) }
