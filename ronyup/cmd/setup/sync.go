package setup

import (
	"github.com/clubpay/ronykit/ronyup/internal/scaffold"

	"github.com/spf13/cobra"
)

// Re-export sync constants for the interactive flow and command-level tests.
const (
	syncSectionAgents   = scaffold.SyncSectionAgents
	syncSectionAI       = scaffold.SyncSectionAI
	syncSectionHooks    = scaffold.SyncSectionHooks
	syncSectionDevops   = scaffold.SyncSectionDevops
	syncSectionDocs     = scaffold.SyncSectionDocs
	syncSectionSkills   = scaffold.SyncSectionSkills
	syncSectionBackend  = scaffold.SyncSectionBackend
	syncSectionFrontend = scaffold.SyncSectionFrontend
	syncSectionAll      = scaffold.SyncSectionAll
	syncKindAuto        = scaffold.SyncKindAuto
	skillSyncInstalled  = scaffold.SkillSyncInstalled
)

var syncOpt = struct {
	RepoDir    string
	Kind       string
	Only       []string
	Overwrite  bool
	SkillsMode []string
}{}

func init() {
	flags := CmdSetupSync.Flags()
	flags.StringVarP(
		&syncOpt.RepoDir,
		"repoDir",
		"r",
		".",
		"repository root to sync (default: current directory)",
	)
	flags.StringVarP(
		&syncOpt.Kind,
		"kind",
		"k",
		syncKindAuto,
		"workspace kind: auto | backend | fullstack | frontend",
	)
	flags.StringSliceVarP(
		&syncOpt.Only,
		"only",
		"o",
		[]string{syncSectionAll},
		"sections to refresh: all | agents | ai | hooks | devops | docs | skills | backend | frontend",
	)
	flags.BoolVar(
		&syncOpt.Overwrite,
		"overwrite",
		false,
		"replace existing scaffold files (default: add missing files only)",
	)
	flags.StringSliceVarP(
		&syncOpt.SkillsMode,
		"skills",
		"s",
		[]string{skillSyncInstalled},
		"skills sync mode: installed (update skills already in .agents/skills) | default | all | none | <skill-id>",
	)
	flags.StringVarP(
		&opt.ApplicationName,
		"appName",
		"a",
		"",
		"application name for templates (default: derived from module or directory)",
	)

	_ = CmdSetupSync.RegisterFlagCompletionFunc(
		"kind",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{
				syncKindAuto,
				scaffold.KindBackend,
				scaffold.KindFullstack,
				scaffold.KindFrontend,
			}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	_ = CmdSetupSync.RegisterFlagCompletionFunc(
		"only",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return append(
				[]string{syncSectionAll},
				scaffold.SyncSections...,
			), cobra.ShellCompDirectiveNoFileComp
		},
	)

	Cmd.AddCommand(CmdSetupSync)
}

var CmdSetupSync = &cobra.Command{
	Use:   "sync",
	Short: "Refresh scaffolded workspace boilerplate from the embedded skeleton",
	Long: `Update scaffold-managed files in an existing workspace (AGENTS.md, devops/, hooks,
skills, Makefiles, etc.) without touching application code under cmd/, feature/, or pkg/.

By default only missing scaffold files are added. Pass --overwrite to replace files that
already exist. Run from the repository root, or from backend/ in a fullstack workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runSync(cmd)
	},
}

func runSync(cmd *cobra.Command) error {
	req := scaffold.SyncRequest{
		StartDir:   syncOpt.RepoDir,
		Kind:       syncOpt.Kind,
		Only:       syncOpt.Only,
		Overwrite:  syncOpt.Overwrite,
		SkillsMode: syncOpt.SkillsMode,
		AppName:    opt.ApplicationName,
	}

	if f := cmd.Flag("repoModule"); f != nil && f.Changed {
		req.Module = opt.RepositoryGoModule
	}

	return scaffold.SyncWorkspace(req, cmd)
}
