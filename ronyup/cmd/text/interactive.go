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
	var langsStr string
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
		),
	).Run()
	if err != nil {
		return err
	}

	if langsStr != "" {
		opt.Languages = strings.Split(langsStr, ",")
		for i := range opt.Languages {
			opt.Languages[i] = strings.TrimSpace(opt.Languages[i])
		}
	}

	fmt.Printf("Extracting translations for %v...\n", opt.Languages)

	return nil
}
