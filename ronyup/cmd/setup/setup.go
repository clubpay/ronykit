package setup

import (
	"github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/spf13/cobra"
)

// Re-export workspace kinds for interactive/tests.
const (
	KindBackend   = scaffold.KindBackend
	KindFullstack = scaffold.KindFullstack
	KindFrontend  = scaffold.KindFrontend
)

const frontendDir = "frontend"

// TemplateInput is the scaffold template context (re-exported for migrate/bundle).
type TemplateInput = scaffold.TemplateInput

var opt = struct {
	ApplicationName        string
	RepositoryRootDir      string
	RepositoryGoModule     string
	Kind                   string
	FeatureContainerFolder string
	FeatureDir             string
	FeatureName            string
	Force                  bool
	Template               string
	GroupByTemplate        bool
	Custom                 map[string]string
	Skills                 []string
}{}

func init() {
	rootFlagSet := Cmd.PersistentFlags()
	rootFlagSet.StringVarP(
		&opt.RepositoryGoModule,
		"repoModule",
		"m",
		"github.com/your/repo",
		"go module for the repository",
	)
	rootFlagSet.BoolVarP(
		&opt.Force,
		"force",
		"f",
		false,
		"replace a non-empty destination directory before setup",
	)
	rootFlagSet.StringToStringVarP(
		&opt.Custom,
		"custom",
		"c",
		map[string]string{},
		"custom values for the template",
	)

	featureFlagSet := CmdSetupFeature.Flags()
	featureFlagSet.StringVarP(
		&opt.FeatureContainerFolder,
		"featurePrefix",
		"",
		scaffold.DefaultFeaturePrefix,
		"prefix for feature folder",
	)
	featureFlagSet.StringVarP(
		&opt.FeatureDir,
		"featureDir",
		"p",
		"my_feature",
		"destination directory inside the Go workspace for the feature",
	)
	featureFlagSet.StringVarP(
		&opt.FeatureName,
		"featureName",
		"n",
		"myfeature",
		"feature name",
	)
	featureFlagSet.StringVarP(
		&opt.Template,
		"template",
		"t",
		"service",
		"possible values: service | job | gateway",
	)
	featureFlagSet.BoolVarP(
		&opt.GroupByTemplate,
		"groupByTemplate",
		"g",
		false,
		"group features by template",
	)

	_ = CmdSetupFeature.RegisterFlagCompletionFunc(
		"template",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return scaffold.FeatureTemplates, cobra.ShellCompDirectiveNoFileComp
		},
	)

	workspaceFlagSet := CmdSetupWorkspace.Flags()
	workspaceFlagSet.StringVarP(
		&opt.RepositoryRootDir,
		"repoDir",
		"r",
		"./my-repo",
		"destination directory for the setup",
	)
	workspaceFlagSet.StringVarP(
		&opt.ApplicationName,
		"appName",
		"a",
		"myapp",
		"application name",
	)
	workspaceFlagSet.StringVarP(
		&opt.Kind,
		"kind",
		"k",
		KindBackend,
		"workspace kind: backend | fullstack | frontend",
	)
	workspaceFlagSet.StringSliceVarP(
		&opt.Skills,
		"skills",
		"s",
		nil,
		"agent skills to pre-install (comma-separated IDs, or default | all | none)",
	)

	_ = CmdSetupWorkspace.RegisterFlagCompletionFunc(
		"kind",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return scaffold.WorkspaceKinds, cobra.ShellCompDirectiveNoFileComp
		},
	)

	_ = CmdSetupWorkspace.RegisterFlagCompletionFunc(
		"skills",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			tokens := append(
				[]string{scaffold.SkillTokenDefault, scaffold.SkillTokenAll, scaffold.SkillTokenNone},
				scaffold.AllSkillIDs()...,
			)

			return tokens, cobra.ShellCompDirectiveNoFileComp
		},
	)

	Cmd.AddCommand(CmdSetupWorkspace, CmdSetupFeature)
}

var Cmd = &cobra.Command{
	Use:                "setup",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			return RunInteractive(cmd)
		}

		return nil
	},
}

var CmdSetupWorkspace = &cobra.Command{
	Use:                "workspace",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runWorkspace(cmd)
	},
}

func runWorkspace(cmd *cobra.Command) error {
	return scaffold.SetupWorkspace(cmd.Context(), scaffold.WorkspaceRequest{
		Path:    opt.RepositoryRootDir,
		Module:  opt.RepositoryGoModule,
		AppName: opt.ApplicationName,
		Kind:    opt.Kind,
		Skills:  opt.Skills,
		Force:   opt.Force,
		Custom:  opt.Custom,
	}, cmd)
}

var CmdSetupFeature = &cobra.Command{
	Use:                "feature",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runFeature(cmd)
	},
}

func runFeature(cmd *cobra.Command) error {
	req := scaffold.FeatureRequest{
		StartDir:        rkit.GetCurrentDir(),
		FeatureDir:      opt.FeatureDir,
		FeatureName:     opt.FeatureName,
		Template:        opt.Template,
		FeaturePrefix:   opt.FeatureContainerFolder,
		GroupByTemplate: opt.GroupByTemplate,
		Force:           opt.Force,
	}

	if f := cmd.Flag("repoModule"); f != nil && f.Changed {
		req.Module = opt.RepositoryGoModule
	}

	return scaffold.SetupFeature(cmd.Context(), req, cmd)
}
