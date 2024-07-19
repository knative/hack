package extract_test

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"knative.dev/hack/pkg/inflator/extract"
	"knative.dev/hack/pkg/utest/assert"
	"knative.dev/hack/pkg/utest/require"
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
	errOut := standarizeErrOut(prtr.err.String(), tmpdir)
	assert.Equal(t, prtr.out.String(), tmpdir+"/library.sh\n")
	assert.Equal(t,
		`[hack] Extracting hack scripts to directory: /tmp/x
[hack] boilerplate.go.txt
[hack] codegen-library.sh
[hack] e2e-tests.sh
[hack] infra-library.sh
[hack] library.sh
[hack] microbenchmarks.sh
[hack] performance-tests.sh
[hack] presubmit-tests.sh
[hack] release.sh
[hack] shellcheck-presubmit.sh
`, errOut)

	// second time should be a no-op
	prtr = &testPrinter{}
	err = op.Extract(prtr)
	require.NoError(t, err)
	errOut = standarizeErrOut(prtr.err.String(), tmpdir)
	assert.Equal(t, prtr.out.String(), tmpdir+"/library.sh\n")
	assert.Equal(t,
		`[hack] Extracting hack scripts to directory: /tmp/x
[hack] boilerplate.go.txt             up-to-date
[hack] codegen-library.sh             up-to-date
[hack] e2e-tests.sh                   up-to-date
[hack] infra-library.sh               up-to-date
[hack] library.sh                     up-to-date
[hack] microbenchmarks.sh             up-to-date
[hack] performance-tests.sh           up-to-date
[hack] presubmit-tests.sh             up-to-date
[hack] release.sh                     up-to-date
[hack] shellcheck-presubmit.sh        up-to-date
`, errOut)
}

func standarizeErrOut(errOut string, tmpdir string) string {
	errOut = strings.ReplaceAll(errOut, tmpdir, "/tmp/x")
	re := regexp.MustCompile(`\s+\d+ (?:Ki)?B \+*`)
	errOut = re.ReplaceAllString(errOut, "")
	return errOut
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
