package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wavesoftware/go-commandline"
	"knative.dev/hack/script/extract"
)

// Options to override the commandline for testing purposes.
var Options []commandline.Option //nolint:gochecknoglobals

type App struct{}

func (a App) Command() *cobra.Command {
	fl := &flags{}
	c := &cobra.Command{
		Use:   "script library.sh",
		Short: "Script is a tool for running Hack scripts",
		Long: "Script will extract Hack scripts to a temporary directory, " +
			"and provide a source file path to requested script",
		Example: `
# In Bash script
source "$(go run knative.dev/hack/cmd/script library.sh)"`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			op := createOperation(fl, argv)
			return op.Extract(cmd)
		},
	}
	c.SetOut(os.Stdout)
	return fl.withFlags(c)
}

func createOperation(fl *flags, argv []string) extract.Operation {
	return extract.Operation{
		ScriptName: argv[0],
		Verbose:    fl.verbose,
	}
}
