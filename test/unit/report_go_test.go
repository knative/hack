package unit_test

import (
	"fmt"
	"testing"
)

func TestReportGoTest(t *testing.T) {
	tmpdir := t.TempDir()
	sc := newShellScript(
		loadFile("source-library.bash", "fake-prow-job.bash"),
		instructions(fmt.Sprintf(`export ARTIFACTS="%s"`, tmpdir)),
	)
	logChecks := []check{
		contains("Finished run, return code is 1"),
		contains(fmt.Sprintf("XML report written to %s", tmpdir)),
		contains(fmt.Sprintf("Test log (JSONL) written to %s", tmpdir)),
		contains(fmt.Sprintf("Test log (ANSI) written to %s", tmpdir)),
		contains(fmt.Sprintf("Test log (HTML) written to %s", tmpdir)),
	}
	tcs := []testCase{{
		name: `report_go_test -tags=library -run TestFailsWithFatal ./test`,
		stdout: append([]check{
			contains("=== RUN   TestFailsWithFatal"),
			contains("fatal\tTestFailsWithFatal\tlibrary_test.go:48\tFailed with logger.Fatal()"),
			contains("FAIL test.TestFailsWithFatal"),
		}, logChecks...),
		stderr: lines("exit status 1"),
	}, {
		name: `report_go_test -tags=library -run TestFailsWithPanic ./test`,
		stdout: append([]check{
			contains("=== RUN   TestFailsWithPanic"),
			contains("panic: test timed out after 5m0s"),
			contains("FAIL test.TestFailsWithPanic"),
		}, logChecks...),
		stderr: lines("exit status 1"),
	}, {
		name: `report_go_test -tags=library -run TestFailsWithSigQuit ./test`,
		stdout: append([]check{
			contains("=== RUN   TestFailsWithSigQuit"),
			contains("SIGQUIT: quit"),
			contains("FAIL test.TestFailsWithSigQuit"),
		}, logChecks...),
		stderr: lines("exit status 1"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
