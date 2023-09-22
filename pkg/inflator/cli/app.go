package cli

import (
	"fmt"

	"knative.dev/hack/pkg/inflator/extract"
	"knative.dev/hack/pkg/retcode"
)

// Execute will execute the application.
func Execute(opts []Option) Result {
	ex := Execution{}.Default().Configure(opts)
	fl, err := parseArgs(&ex)
	if err != nil {
		return Result{
			Execution: ex,
			Err:       err,
		}
	}
	op := createOperation(fl, ex.Args)
	return Result{
		Execution: ex,
		Err:       op.Extract(ex),
	}
}

// ExecuteOrDie will execute the application or perform os.Exit in case of error.
func ExecuteOrDie(opts ...Option) {
	if r := Execute(opts); r.Err != nil {
		r.PrintErrln(fmt.Sprintf("%v", r.Err))
		r.Exit(retcode.Calc(r.Err))
	}
}

type usageErr struct{}

func (u usageErr) Error() string {
	return `Hacks as Go self-extracting binary

Will extract Hack scripts to a temporary directory, and provide a source
file path to requested shell script.

# In Bash script
source "$(go run knative.dev/hack/cmd/script@latest library.sh)"

Usage:
	script [flags] library.sh

Flags:
	-h, --help      help
	-v, --verbose   verbose output
`
}

func createOperation(fl *flags, argv []string) extract.Operation {
	return extract.Operation{
		ScriptName: argv[0],
		Verbose:    fl.verbose,
	}
}
