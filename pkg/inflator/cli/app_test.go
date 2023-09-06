package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"knative.dev/hack/pkg/inflator/cli"
	"knative.dev/hack/pkg/inflator/extract"
)

func TestApp(t *testing.T) {
	tmpdir := t.TempDir()
	t.Setenv(extract.HackScriptsDirEnvVar, tmpdir)
	t.Setenv(cli.ManualVerboseEnvVar, "true")
	c := cli.App{}.Command()
	var (
		outb bytes.Buffer
		errb bytes.Buffer
	)
	c.SetOut(&outb)
	c.SetErr(&errb)
	c.SetArgs([]string{"e2e-tests.sh"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, outb.String(), tmpdir+"/e2e-tests.sh\n")
	assert.Equal(t, errb.String(), "")
}
