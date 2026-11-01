package z

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckPath verifies that dest is below root and that no existing path
// component below root is a symlink. The root itself may be a symlink chosen
// by the caller.
func CheckPath(root, dest string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(absRoot, absDest)
	if err != nil {
		return err
	}

	if rel == "." {
		return nil
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("destination %s escapes root %s", dest, root)
	}

	current := absRoot

	parts := strings.Split(rel, string(filepath.Separator))
	for i, part := range parts {
		current = filepath.Join(current, part)

		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination path contains symlink %s", current)
		}

		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("destination parent %s is not a directory", current)
		}
	}

	return nil
}
