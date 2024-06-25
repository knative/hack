package unit_test

import "testing"

func TestCodegen(t *testing.T) {
	t.Parallel()
	sc := newShellScript(
		loadFile("source-codegen-library.bash"),
		mockGo(),
	)
	tcs := []testCase{{
		name: "generate-groups deepcopy " +
			"knative.dev/hack/test/e2e/apis/hack/v1alpha1/generated " +
			"knative.dev/hack/test/e2e/apis " +
			"hack:v1alpha1",
	}}
	for i := range tcs {
		tc := tcs[i]
		t.Run(tc.name, tc.test(sc))
	}
}
