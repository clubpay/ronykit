package setup

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

// selectSkillsInteractive presents a multi-select of bundled agent skills with
// the workspace-kind defaults pre-checked, and records the choice in opt.Skills
// so runWorkspace resolves and installs exactly what the user picked.
func selectSkillsInteractive() error {
	defaults := map[string]bool{}
	for _, id := range defaultSkillIDs(opt.Kind) {
		defaults[id] = true
	}

	options := make([]huh.Option[string], 0, len(skillCatalog))
	for _, s := range skillCatalog {
		label := fmt.Sprintf("[%s] %s — %s", s.Category, s.Name, s.Description)
		options = append(options, huh.NewOption(label, s.ID).Selected(defaults[s.ID]))
	}

	var selected []string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Agent Skills").
				Description("Pre-install agent skills into .agents/skills (space to toggle)").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return err
	}

	// Record the explicit choice. An empty selection must mean "none" so
	// runWorkspace does not fall back to the kind defaults.
	if len(selected) == 0 {
		opt.Skills = []string{skillTokenNone}
	} else {
		opt.Skills = selected
	}

	return nil
}

func RunInteractive(cmd *cobra.Command) error {
	var action string

	err := huh.NewSelect[string]().
		Title("What would you like to do?").
		Options(
			huh.NewOption("Setup a new workspace", "workspace"),
			huh.NewOption("Add a new feature to existing workspace", "feature"),
		).
		Value(&action).
		Run()
	if err != nil {
		return err
	}

	switch action {
	case "workspace":
		return runWorkspaceInteractive(cmd)
	case "feature":
		return runFeatureInteractive(cmd)
	}

	return nil
}

func runWorkspaceInteractive(cmd *cobra.Command) error {
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Repository Directory").
				Description("Where should the workspace be created?").
				Placeholder("./my-repo").
				Value(&opt.RepositoryRootDir),
			huh.NewInput().
				Title("Repository Go Module").
				Description("What is the go module name?").
				Placeholder("github.com/your/repo").
				Value(&opt.RepositoryGoModule),
			huh.NewInput().
				Title("Application Name").
				Description("What is the application name?").
				Placeholder("myapp").
				Value(&opt.ApplicationName),
			huh.NewSelect[string]().
				Title("Workspace Kind").
				Description("Backend only, or backend + frontend split").
				Options(
					huh.NewOption("Backend only", KindBackend),
					huh.NewOption("Backend + Frontend", KindFullstack),
				).
				Value(&opt.Kind),
		),
	).Run()
	if err != nil {
		return err
	}

	if err := selectSkillsInteractive(); err != nil {
		return err
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Creating workspace").
				Description("Creating workspace in " + opt.RepositoryRootDir + "..."),
		),
	).Run()
	if err != nil {
		return err
	}

	return runWorkspace(cmd)
}

func runFeatureInteractive(cmd *cobra.Command) error {
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Feature Parent Directory").
				Description("Parent Directory of the Feature").
				Placeholder("feature").
				Value(&opt.FeatureContainerFolder),
			huh.NewInput().
				Title("Feature Directory").
				Description("Destination directory inside repoDir for the setup").
				Placeholder("auth").
				Value(&opt.FeatureDir),
			huh.NewInput().
				Title("Feature Name").
				Description("Name of the feature").
				Placeholder("auth").
				Value(&opt.FeatureName),
			huh.NewSelect[string]().
				Title("Template").
				Description("Choose a template for the feature").
				Options(
					huh.NewOption("Service", "service"),
					huh.NewOption("Job", "job"),
					huh.NewOption("Gateway", "gateway"),
				).
				Value(&opt.Template),
			huh.NewConfirm().
				Title("Group by Template").
				Affirmative("YES").
				Negative("NO").
				Value(&opt.GroupByTemplate),
		),
	).Run()
	if err != nil {
		return err
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Creating feature").
				Description("Creating feature " + opt.FeatureName + " in " + opt.FeatureDir + "..."),
		),
	).Run()
	if err != nil {
		return err
	}

	return runFeature(cmd)
}
