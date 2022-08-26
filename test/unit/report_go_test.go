package unit_test

import "testing"

func TestReportGoTest(t *testing.T) {
	sc := newShellScript(loadFile(
		"source-library.bash",
		"fake-prow-job.bash",
	))
	tcs := []testCase{{
		name: `report_go_test -tags=library -run TestFailsWithFatal ./test`,
		stdout: []check{
			contains("=== RUN   TestFailsWithFatal"),
			contains("fatal\tTestFailsWithFatal\tlibrary_test.go:48\tFailed with logger.Fatal()"),
			contains("FAIL test.TestFailsWithFatal"),
			contains("Finished run, return code is 1"),
			contains("XML report written"),
			contains("Test log (JSONL) written to"),
			contains("Test log (ANSI) written to"),
			contains("Test log (HTML) written to"),
		},
		stderr: lines("exit status 1"),
	}, {
		name: `report_go_test -tags=library -run TestFailsWithPanic ./test`,
		stdout: []check{
			contains("=== RUN   TestFailsWithPanic"),
			contains("panic: test timed out after 5m0s"),
			contains("FAIL test.TestFailsWithPanic"),
			contains("Finished run, return code is 1"),
			contains("XML report written"),
			contains("Test log (JSONL) written to"),
			contains("Test log (ANSI) written to"),
			contains("Test log (HTML) written to"),
		},
		stderr: lines("exit status 1"),
	}, {
		name: `report_go_test -tags=library -run TestFailsWithSigQuit ./test`,
		stdout: []check{
			contains("=== RUN   TestFailsWithSigQuit"),
			contains("SIGQUIT: quit"),
			contains("FAIL test.TestFailsWithSigQuit"),
			contains("Finished run, return code is 1"),
			contains("XML report written"),
			contains("Test log (JSONL) written to"),
			contains("Test log (ANSI) written to"),
			contains("Test log (HTML) written to"),
		},
		stderr: lines("exit status 1"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
