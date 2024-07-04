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
		mockGo(),
		mockKubectl(),
	)
	tcs := []testCase{{
		name:   `initialize --smoke-test-custom-flag`,
		stdout: []check{subheader("Custom flag detected")},
	}, {
		name: `fail_test`,
		commands: []string{
			`initialize --run-tests`,
			`fail_test`,
		},
		retcode: retcode(111),
		stdout: []check{
			contains("Cluster is gke_deadbeef_1.24"),
			subheader("Dumping the cluster state"),
		},
		stderr: aborted("test failed"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
