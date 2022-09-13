package unit_test

import "testing"

func TestE2eHelpers(t *testing.T) {
	t.Parallel()
	sc := newShellScript(
		fakeProwJob(),
		loadFile(
			"source-e2e-tests.bash",
			"smoke-test-custom-flag.bash",
			"fake-dumps.bash",
		),
	)
	tcs := []testCase{{
		name:   `initialize --smoke-test-custom-flag`,
		stdout: lines(">> All tests passed"),
	}, {
		name: `fail_test`,
		commands: []string{
			`initialize --run-test true`,
			`fail_test`,
		},
		stderr:  aborted("test failed"),
		retcode: retcode(111),
		stdout: []check{
			contains(">> DUMPING THE CLUSTER STATE"),
			contains(">> STARTING KUBE PROXY"),
			contains(">> GRABBING K8S METRICS"),
		},
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
