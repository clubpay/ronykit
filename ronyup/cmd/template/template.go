package template

import (
	"path/filepath"

	"github.com/clubpay/ronykit/x/rkit"
	"github.com/spf13/cobra"
)

var opt = struct {
	TemplateRepo string
	Template     string
	Custom       map[string]string
}{}

func init() {
	rootFlagSet := Cmd.PersistentFlags()
	rootFlagSet.StringToStringVarP(
		&opt.Custom,
		"custom",
		"c",
		map[string]string{},
		"custom values for the template",
	)

	generateFlagSet := CmdGenerate.Flags()
	generateFlagSet.StringVarP(
		&opt.TemplateRepo,
		"templateRepo",
		"r",
		"",
		"if not set we use the standard templates repo embedded into the binary",
	)
	generateFlagSet.StringVarP(
		&opt.Template,
		"template",
		"t",
		"service",
		"possible values: dto | dao | serializer",
	)

	_ = CmdGenerate.RegisterFlagCompletionFunc(
		"template",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"service", "job", "gateway"}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	Cmd.AddCommand(CmdGenerate)
}

var Cmd = &cobra.Command{
	Use:                "template",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		//if len(args) == 0 && cmd.Flags().NFlag() == 0 {
		//	return RunInteractive()
		//}

		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		pkgs, err := ParseGoPath(rkit.Must(filepath.Abs(".")))
		if err != nil {
			return err
		}

		cmd.Println("packages:")
		for _, pkg := range pkgs {
			cmd.Println(pkg.Name)
			cmd.Println("files:", len(pkg.Files))
			cmd.Println("types:", len(pkg.Types))
			for _, f := range pkg.Functions {
				cmd.Println("\t", f.Name, f.Receiver, f.ReceiverType, f.Params, f.Results)
			}
			cmd.Println("comments", len(pkg.Comments))
		}

		return nil
	},
}

var CmdGenerate = &cobra.Command{
	Use: "generate",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
