package cli

import (
	"github.com/spf13/cobra"
)

type flags struct {
	verbose bool
}

func (f *flags) withFlags(c *cobra.Command) *cobra.Command {
	fl := c.PersistentFlags()
	fl.BoolVarP(&f.verbose, "verbose", "v", false, "Print verbose output on Stderr")
	return c
}
