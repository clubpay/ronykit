package cmd

import (
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/clubpay/ronykit/boxship"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use: "boxship",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		updater, err := selfupdate.NewUpdater(selfupdate.Config{})
		if err != nil {
			return err
		}
		latest, found, err := updater.DetectLatest(repoSlug)
		if err != nil {
			return err
		}

		if !found || boxship.Version.GT(semver.MustParse(latest.Version.String())) {
			return nil
		}

		cmd.Println("## UPDATE ## there is new version to download, type: 'boxship update' to update your boxship")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {},
}

type errorHandler struct{}

func (e errorHandler) HandleError(err error) {
	fmt.Println(err)
}
