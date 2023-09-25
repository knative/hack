package cli_test

import (
	"bytes"
	"testing"

	"knative.dev/hack/pkg/inflator/cli"
	"knative.dev/hack/pkg/inflator/extract"
	"knative.dev/hack/pkg/utest/assert"
	"knative.dev/hack/pkg/utest/require"
)

func TestExecute(t *testing.T) {
	tmpdir := t.TempDir()
	t.Setenv(extract.HackScriptsDirEnvVar, tmpdir)
	t.Setenv(cli.ManualVerboseEnvVar, "true")
	var (
		outb bytes.Buffer
		errb bytes.Buffer
	)

	r := cli.Execute([]cli.Option{func(ex *cli.Execution) {
		ex.Args = []string{"e2e-tests.sh"}
		ex.Stdout = &outb
		ex.Stderr = &errb
	}})

	require.NoError(t, r.Err)
	assert.Equal(t, outb.String(), tmpdir+"/e2e-tests.sh\n")
	assert.Equal(t, errb.String(), "")
}
