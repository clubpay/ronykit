package cmd

import (
	"fmt"
	"os"

	"github.com/blang/semver/v4"
	"github.com/clubpay/ronykit/boxship"
	"github.com/joho/godotenv"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const repoSlug = "ronaksoft/boxship"

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates boxship",
	Long: `
Updates boxship to the latest version. You need to have a GitHub Personal Access
Token (GITHUB_PAT) in your environment.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = godotenv.Load()
		updater, err := selfupdate.NewUpdater(selfupdate.Config{
			APIToken: os.Getenv("GITHUB_PAT"),
		})
		if err != nil {
			return err
		}
		latest, found, err := updater.DetectLatest(repoSlug)
		if err != nil {
			return err
		}
		execCmd, err := os.Executable()
		if err != nil {
			return err
		}

		cmd.Println("Current:", boxship.Version.String())
		cmd.Println("Latest:", semver.MustParse(latest.Version.String()))
		if !found || boxship.Version.GTE(semver.MustParse(latest.Version.String())) {
			cmd.Println("You are using the latest version.")

			return nil
		}

		cmd.Println(fmt.Sprintf("We are updating boxship ... %s -> %s", boxship.Version.String(), latest.Version.String()))
		err = updater.UpdateTo(latest, execCmd)
		if err != nil {
			return err
		}

		cmd.Println("Successfully updated to version:", latest.Version)
		cmd.Println("Release Note:")
		cmd.Println(latest.ReleaseNotes)

		return nil
	},
}
