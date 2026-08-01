package setup

import (
	"github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/spf13/cobra"
)

var migrateOpt = struct {
	DryRun bool
}{}

var CmdSetupMigrate = &cobra.Command{
	Use:   "migrate",
	Short: "Upgrade an existing workspace to a newer scaffold layout",
}

var CmdSetupMigrateBundles = &cobra.Command{
	Use:   "bundles",
	Short: "Migrate a legacy workspace to the bundle + pkg/runner layout",
	Long: `Upgrade workspaces created before executable bundles were introduced.

The command is idempotent: safe to run multiple times. It will:

  - add pkg/runner/ (shared bootstrap) when missing or outdated
  - rewrite cmd/all-in-one/main.go to delegate to pkg/runner
  - remove legacy cmd/all-in-one/middleware.go and healthz.go (or the same under cmd/service/)
  - rename legacy cmd/service/ to cmd/all-in-one/ when present
  - remove legacy internal/runner/ and cmd/runner/ when present
  - create bundles.yaml when missing (default all-in-one bundle uses "*")
  - register pkg/runner in go.work and refresh bundle features.go files

Run from the Go workspace root (directory with go.work) or from the repository
root in a fullstack workspace (where go.work lives under backend/).

Examples:
  ronyup setup migrate bundles
  ronyup setup migrate bundles --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runMigrateBundles(cmd)
	},
}

func init() {
	flags := CmdSetupMigrateBundles.Flags()
	flags.BoolVar(&migrateOpt.DryRun, "dry-run", false, "print planned changes without writing files")

	CmdSetupMigrate.AddCommand(CmdSetupMigrateBundles)
	Cmd.AddCommand(CmdSetupMigrate)
}

func runMigrateBundles(cmd *cobra.Command) error {
	req := scaffold.MigrateBundlesRequest{
		StartDir: rkit.GetCurrentDir(),
		AppName:  opt.ApplicationName,
		DryRun:   migrateOpt.DryRun,
	}

	if f := cmd.Flag("repoModule"); f != nil && f.Changed {
		req.Module = opt.RepositoryGoModule
	}

	return scaffold.MigrateBundles(cmd.Context(), req, cmd)
}
