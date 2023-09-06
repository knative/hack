package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	// ManualVerboseEnvVar is the environment variable that can be set to disable
	// automatic verbose mode on CI servers.
	ManualVerboseEnvVar = "KNATIVE_HACK_SCRIPT_MANUAL_VERBOSE"
)

type flags struct {
	verbose bool
}

func (f *flags) withFlags(c *cobra.Command) *cobra.Command {
	fl := c.PersistentFlags()
	fl.BoolVarP(&f.verbose, "verbose", "v", isCiServer(), "Print verbose output on Stderr")
	return c
}

func isCiServer() bool {
	if strings.HasPrefix(strings.ToLower(os.Getenv(ManualVerboseEnvVar)), "t") {
		return false
	}
	return os.Getenv("CI") != "" ||
		os.Getenv("BUILD_ID") != "" ||
		os.Getenv("PROW_JOB_ID") != ""
}
