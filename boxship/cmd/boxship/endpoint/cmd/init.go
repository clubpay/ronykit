package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/clubpay/ronykit/boxship/pkg/settings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/clubpay/ronykit/boxship/templates"
	"github.com/gobuffalo/genny/v2"
	"github.com/spf13/cobra"
)

func init() {
	InitCmd.Flags().String(settings.Template, "default1", "possible values: [default1]")
	InitCmd.Flags().String(settings.OutputDir, "boxship-devenv", "")
	InitCmd.Flags().Bool(settings.Force, false, "force recreate everything")
}

var InitCmd = &cobra.Command{
	Use:   "init <target>",
	Short: "initialize the development environment",
	Long: `
You need to run init command only once you first want to setup your environment. If you run the init command
multiple times with same folder it overwrites all the changes that you have made to your setup files.

There are a few templates that you can use to setup your environment, currently the following are supported:
1. default1:
	This template is suitable is good starting point for setup your local development environment

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Println("usage: boxship init <yourDestinationFolder>")
			cmd.Println("example: boxship init default1")

			return nil
		}
		output := args[0]

		template, err := cmd.Flags().GetString(settings.Template)
		if err != nil {
			return err
		}
		switch template {
		case "default1":
		default:
			return fmt.Errorf("not supported template")
		}

		pathPrefix := fmt.Sprintf("default/%s", template)
		g := genny.New()
		g.File(genny.NewDir(output, os.ModeDir|os.ModePerm))

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

				g.File(genny.NewFileB(filepath.Join(output, relPath), data))

				return nil
			},
		)
		if err != nil {
			return err
		}

		r := genny.WetRunner(context.Background())
		err = r.Chdir(output, func() error {
			if err = r.With(g); err != nil {
				return err
			}

			if err = r.Run(); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}

		path := filepath.Join(output, "README.MD")
		source, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		result := markdown.Render(string(source), 80, 6)
		cmd.Println(string(result))

		return nil
	},
}
