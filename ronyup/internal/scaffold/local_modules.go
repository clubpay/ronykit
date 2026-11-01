package scaffold

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/clubpay/ronykit/ronyup/internal/z"
)

func addLocalModuleReplacement(
	ctx context.Context,
	goRoot, moduleDir, repoModule, packagePath string,
) error {
	if err := validateRelativePath("local module path", packagePath); err != nil {
		return err
	}

	localDir := filepath.Join(goRoot, filepath.FromSlash(packagePath))
	if err := z.CheckPath(goRoot, localDir); err != nil {
		return err
	}

	rel, err := filepath.Rel(moduleDir, localDir)
	if err != nil {
		return err
	}

	modulePath := path.Join(repoModule, packagePath)
	replace := "-replace=" + modulePath + "=" + filepath.ToSlash(rel)

	require := "-require=" + modulePath + "@v0.0.0"
	if err := z.RunCmd(
		ctx,
		z.RunCmdParams{Dir: moduleDir},
		"go",
		"mod",
		"edit",
		require,
		replace,
	); err != nil {
		return fmt.Errorf("use local module %s: %w", modulePath, err)
	}

	return nil
}
