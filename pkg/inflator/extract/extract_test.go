package extract_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"knative.dev/hack/pkg/inflator/extract"
)

func TestExtract(t *testing.T) {
	tmpdir := t.TempDir()
	t.Setenv(extract.HackScriptsDirEnvVar, tmpdir)
	op := extract.Operation{
		ScriptName: "library.sh",
		Verbose:    true,
	}
	prtr := &testPrinter{}
	err := op.Extract(prtr)
	require.NoError(t, err)
	assert.Equal(t, prtr.out.String(), tmpdir+"/library.sh\n")
	assert.Equal(t,
		`[hack] Extracting hack scripts to directory: /tmp/x
[hack] codegen-library.sh               1 KiB +
[hack] e2e-tests.sh                     6 KiB ++
[hack] infra-library.sh                 5 KiB +
[hack] library.sh                      33 KiB +++++++
[hack] microbenchmarks.sh               2 KiB +
[hack] performance-tests.sh             6 KiB ++
[hack] presubmit-tests.sh              12 KiB +++
[hack] release.sh                      27 KiB ++++++
[hack] shellcheck-presubmit.sh          1 KiB +
`, strings.ReplaceAll(prtr.err.String(), tmpdir, "/tmp/x"))

	// second time should be a no-op
	prtr = &testPrinter{}
	err = op.Extract(prtr)
	require.NoError(t, err)
	assert.Equal(t, prtr.out.String(), tmpdir+"/library.sh\n")
	assert.Equal(t,
		`[hack] Extracting hack scripts to directory: /tmp/x
[hack] codegen-library.sh             up-to-date
[hack] e2e-tests.sh                   up-to-date
[hack] infra-library.sh               up-to-date
[hack] library.sh                     up-to-date
[hack] microbenchmarks.sh             up-to-date
[hack] performance-tests.sh           up-to-date
[hack] presubmit-tests.sh             up-to-date
[hack] release.sh                     up-to-date
[hack] shellcheck-presubmit.sh        up-to-date
`, strings.ReplaceAll(prtr.err.String(), tmpdir, "/tmp/x"))
}

type testPrinter struct {
	out bytes.Buffer
	err bytes.Buffer
}

func (t *testPrinter) Print(i ...interface{}) {
	_, _ = fmt.Fprint(&t.out, i...)
}

func (t *testPrinter) Println(i ...interface{}) {
	t.Print(fmt.Sprintln(i...))
}

func (t *testPrinter) Printf(format string, i ...interface{}) {
	t.Print(fmt.Sprintf(format, i...))
}

func (t *testPrinter) PrintErr(i ...interface{}) {
	_, _ = fmt.Fprint(&t.err, i...)
}

func (t *testPrinter) PrintErrln(i ...interface{}) {
	t.PrintErr(fmt.Sprintln(i...))
}

func (t *testPrinter) PrintErrf(format string, i ...interface{}) {
	t.PrintErr(fmt.Sprintf(format, i...))
}
