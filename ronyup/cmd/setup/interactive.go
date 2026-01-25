package setup

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func RunInteractive() error {
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
		return runWorkspaceInteractive()
	case "feature":
		return runFeatureInteractive()
	}

	return nil
}

func runWorkspaceInteractive() error {
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
		),
	).Run()
	if err != nil {
		return err
	}

	fmt.Printf("Creating workspace in %s...\n", opt.RepositoryRootDir)

	return nil
}

func runFeatureInteractive() error {
	err := huh.NewForm(
		huh.NewGroup(
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
		),
	).Run()
	if err != nil {
		return err
	}

	fmt.Printf("Creating feature %s in %s...\n", opt.FeatureName, opt.FeatureDir)

	return nil
}
