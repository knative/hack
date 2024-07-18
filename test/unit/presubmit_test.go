package unit_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thanhpk/randstr"
)

func TestPresubmitTestMainFunc(t *testing.T) {
	sc := newShellScript(
		mockDeepcopyGen(t),
		fakeProwJob(),
		loadFile("source-presubmit-tests.bash"),
		mockGo(),
		mockKubectl(),
	)
	tcs := []testCase{{
		name: "main --build-tests",
		stdout: []check{
			header("RUNNING BUILD TESTS"),
			contains("Build tests for knative.dev/hack/test"),
			contains("Build tests for knative.dev/hack/schema"),
			contains("Build tests for knative.dev/hack"),
			contains("Checking that go code builds"),
			contains("👻 go test -vet=off -tags e2e,library -exec echo ./..."),
			contains("👻 go test -vet=off -tags  -exec echo ./..."),
			contains("👻 go run knative.dev/test-infra/tools/kntest/cmd/kntest@latest" +
				" junit --suite=_build_tests --name=Check_Licenses --err-msg= --dest="),
			header("BUILD TESTS PASSED"),
		},
	}, {
		name: "main --unit-tests",
		stdout: []check{
			header("RUNNING UNIT TESTS"),
			contains("Unit tests for knative.dev/hack/test"),
			contains("Unit tests for knative.dev/hack/schema"),
			contains("Unit tests for knative.dev/hack"),
			contains("Running go test with args: -short -race -count 1 ./..."),
			contains("👻 go run gotest.tools/gotestsum@v1.11.0 --format testname --junitfile"),
			contains("-- -short -race -count 1 ./..."),
			header("UNIT TESTS PASSED"),
		},
	}}
	for i := range tcs {
		tc := tcs[i]
		t.Run(tc.name, tc.test(sc))
	}
}

func TestPrType(t *testing.T) {
	t.Parallel()
	sc := newShellScript(
		fakeProwJob(),
		loadFile("source-presubmit-tests.bash"),
		mockGo(),
	)
	tcs := []testCase{{
		name: "PR-type-docs",
		commands: []string{
			listChangedFiles("README.md", "OWNERS", "foo.png"),
			"initialize_environment",
			`echo ":${IS_DOCUMENTATION_PR}:${IS_PRESUBMIT_EXEMPT_PR}:"`,
		},
		stdout: []check{contains(":1:0:")},
	}, {
		name: "PR-type-OWNERS-README-go",
		commands: []string{
			listChangedFiles("OWNERS", "README.md", "foo.go"),
			"initialize_environment",
			`echo ":${IS_DOCUMENTATION_PR}:${IS_PRESUBMIT_EXEMPT_PR}:"`,
		},
		stdout: []check{contains(":0:0:")},
	}, {
		name: "PR-type-OWNERS-README",
		commands: []string{
			listChangedFiles("OWNERS", "README.md"),
			"initialize_environment",
			`echo ":${IS_DOCUMENTATION_PR}:${IS_PRESUBMIT_EXEMPT_PR}:"`,
		},
		stdout: []check{contains(":1:0:")},
	}, {
		name: "PR-type-OWNERS-AUTHORS",
		commands: []string{
			listChangedFiles("OWNERS", "AUTHORS"),
			"initialize_environment",
			`echo ":${IS_DOCUMENTATION_PR}:${IS_PRESUBMIT_EXEMPT_PR}:"`,
		},
		stdout: []check{contains(":0:1:")},
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestCustomAndMultiScript(t *testing.T) {
	t.Parallel()
	rng1 := randstr.String(12)
	rng2 := randstr.String(12)
	sc := newShellScript(
		fakeProwJob(),
		loadFile("source-presubmit-tests.bash"),
		mockGo(),
		mockKubectl(),
	)
	tcs := []testCase{{
		name:     `main --run-test "echo rng"`,
		commands: []string{fmt.Sprintf(`main --run-test "echo %s"`, rng1)},
		stdout:   []check{contains(rng1)},
	}, {
		name:     `main --run-test "echo rng" --run-test "echo rng"`,
		commands: []string{fmt.Sprintf(`main --run-test "echo %s" --run-test "echo %s"`, rng1, rng2)},
		stdout: []check{
			contains(rng1),
			contains(rng2),
		},
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func listChangedFiles(files ...string) string {
	ff := strings.Join(files, "\n")
	return fmt.Sprintf(`function list_changed_files() { echo "%s"; }`, ff)
}
