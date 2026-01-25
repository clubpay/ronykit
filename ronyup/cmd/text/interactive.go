package text

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

func RunInteractive() error {
	var action string

	err := huh.NewSelect[string]().
		Title("What would you like to do?").
		Options(
			huh.NewOption("Extract/Generate translations", "extract"),
		).
		Value(&action).
		Run()
	if err != nil {
		return err
	}

	switch action {
	case "extract":
		return runExtractInteractive()
	}

	return nil
}

func runExtractInteractive() error {
	var (
		langsStr         string
		modulesFilterStr string
	)

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Source Language").
				Placeholder("en-US").
				Value(&opt.SourceLang),
			huh.NewInput().
				Title("Destination Languages").
				Description("Comma separated list of languages").
				Placeholder("en-US,fa-IR,ar-AR").
				Value(&langsStr),
			huh.NewInput().
				Title("Output Directory").
				Placeholder(".").
				Value(&opt.DstDir),
			huh.NewInput().
				Title("Modules").
				Description("Comma separated list of modules to extract, if empty all modules will be extracted.").
				Placeholder("").
				Value(&modulesFilterStr),
			huh.NewInput().
				Title("Gen File").
				Description("Name of the generated file.").
				Placeholder("catalog.go").
				Value(&opt.GenFile),
			huh.NewInput().
				Title("Gen Package Name").
				Description("Name of the generated package.").
				Value(&opt.GenPackageName),
		),
	).Run()
	if err != nil {
		return err
	}

	if langsStr != "" {
		for lang := range strings.SplitSeq(langsStr, ",") {
			opt.Languages = append(opt.Languages, strings.TrimSpace(lang))
		}
	}

	if modulesFilterStr != "" {
		for pkg := range strings.SplitSeq(modulesFilterStr, ",") {
			opt.ModulesFilter = append(opt.ModulesFilter, strings.TrimSpace(pkg))
		}
	}

	fmt.Printf("Extracting translations for %v...\n", opt.Languages)

	return nil
}
