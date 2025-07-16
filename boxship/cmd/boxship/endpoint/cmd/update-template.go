package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/templates"
	"github.com/gobuffalo/genny/v2"
	"github.com/spf13/cobra"
)

func init() {
	UpdateTemplateCmd.Flags().String(settings.Template, "default1", "possible values: [default1]")
	UpdateTemplateCmd.Flags().Bool(settings.Force, false, "force recreate everything")
}

var UpdateTemplateCmd = &cobra.Command{
	Use:   "update-template",
	Short: "updates the setup folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		template, err := cmd.Flags().GetString(settings.Template)
		if err != nil {
			return err
		}
		switch template {
		case "default1":
		default:
			return fmt.Errorf("not supported template")
		}

		_ = os.Rename("./setup", fmt.Sprintf("./setup.backup.%d", time.Now().Unix()))

		pathPrefix := fmt.Sprintf("default/%s", template)
		g := genny.New()
		g.File(genny.NewDir(".", os.ModeDir|os.ModePerm))

		err = fs.WalkDir(
			templates.EnvFolder, pathPrefix,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				data, err := fs.ReadFile(templates.EnvFolder, path)
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(pathPrefix, path)
				if err != nil {
					return err
				}
				g.File(genny.NewFileB(relPath, data))

				return nil
			},
		)
		if err != nil {
			return err
		}

		r := genny.WetRunner(context.Background())
		err = r.With(g)
		if err != nil {
			return err
		}

		err = r.Run()
		if err != nil {
			return err
		}

		return nil
	},
}
