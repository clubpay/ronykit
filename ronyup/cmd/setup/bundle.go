package setup

import (
	"fmt"

	"github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/spf13/cobra"
)

var bundleOpt = struct {
	Name        string
	Services    []string
	Description string
	Gen         bool
	Remove      bool
}{}

var CmdSetupBundle = &cobra.Command{
	Use:   "bundle",
	Short: "Create or refresh executable bundles that mix and match feature services",
	Long: `Bundles declare which feature modules are compiled into each cmd/<name>/ executable.

The default "all-in-one" bundle is the all-in-one dev binary. Additional bundles get their
own cmd/<name>/ module with a selective features.go import list.

Examples:
  ronyup setup bundle --name auth-api --services feature/auth,feature/session
  ronyup setup bundle --gen
  ronyup setup bundle --remove auth-api`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runBundle(cmd)
	},
}

func init() {
	flags := CmdSetupBundle.Flags()
	flags.StringVarP(&bundleOpt.Name, "name", "n", "", "bundle name (cmd/<name>/ directory)")
	flags.StringSliceVarP(
		&bundleOpt.Services,
		"services",
		"s",
		nil,
		"feature module paths (settings.ModuleName), or * for all",
	)
	flags.StringVar(
		&bundleOpt.Description,
		"description",
		"",
		"optional bundle description stored in bundles.yaml",
	)
	flags.BoolVar(&bundleOpt.Gen, "gen", false, "regenerate features.go for every bundle from bundles.yaml")
	flags.BoolVar(
		&bundleOpt.Remove,
		"remove",
		false,
		"remove a bundle from bundles.yaml and delete cmd/<name>/",
	)

	Cmd.AddCommand(CmdSetupBundle)
}

func runBundle(cmd *cobra.Command) error {
	module := ""
	if f := cmd.Flag("repoModule"); f != nil && f.Changed {
		module = opt.RepositoryGoModule
	}

	if bundleOpt.Gen {
		return scaffold.RegenerateBundleFeatures(cmd.Context(), scaffold.RegenerateBundlesRequest{
			StartDir: rkit.GetCurrentDir(),
			Module:   module,
		}, cmd)
	}

	req := scaffold.BundleRequest{
		StartDir:    rkit.GetCurrentDir(),
		Module:      module,
		AppName:     opt.ApplicationName,
		Name:        bundleOpt.Name,
		Services:    bundleOpt.Services,
		Description: bundleOpt.Description,
		Force:       opt.Force,
	}

	if bundleOpt.Remove {
		if req.Name == "" {
			return fmt.Errorf("--name is required with --remove")
		}

		return scaffold.RemoveBundle(cmd.Context(), req, cmd)
	}

	return scaffold.CreateBundle(cmd.Context(), req, cmd)
}
