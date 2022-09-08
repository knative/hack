package unit_test

import (
	"fmt"
	"path"
	"strings"
	"testing"
)

func TestMainFunc(t *testing.T) {
	t.Parallel()
	rootDir := path.Dir(path.Dir(currentDir()))
	sc := newShellScript(
		fakeProwJob(),
		loadFile("source-presubmit-tests.bash"),
		mockGo(),
		mockKubectl(),
	)
	tcs := []testCase{{
		name: `main --build-tests`,
		stdout: []check{
			contains("RUNNING BUILD TESTS"),
			contains("Build tests for knative.dev/hack/test"),
			contains("Build tests for knative.dev/hack/schema"),
			contains("Build tests for knative.dev/hack"),
			contains("Checking that go code builds"),
			contains("go test -vet=off -tags e2e,hack,library -exec echo ./..."),
			contains("go test -vet=off -tags e2e,library -exec echo ./..."),
			contains("go test -vet=off -tags  -exec echo ./..."),
			contains("go run knative.dev/test-infra/tools/kntest/cmd/kntest@latest" +
				" junit --suite=_build_tests --name=Check_Licenses --err-msg= --dest="),
			contains("BUILD TESTS PASSED"),
		},
	}, {
		name: `main --unit-tests`,
		stdout: []check{
			contains("RUNNING UNIT TESTS"),
			contains("Unit tests for knative.dev/hack/test"),
			contains("Unit tests for knative.dev/hack/schema"),
			contains("Unit tests for knative.dev/hack"),
			contains("Running go test with args: -short -race -count 1 ./..."),
			contains("go run gotest.tools/gotestsum@v1.8.0 --format testname --junitfile"),
			contains("-- -short -race -count 1 ./..."),
			contains("UNIT TESTS PASSED"),
		},
	}, {
		name: `main --integration-tests`,
		stdout: []check{
			contains("RUNNING INTEGRATION TESTS"),
			contains("Running integration test test/e2e-tests.sh"),
			contains(fmt.Sprintf("go run knative.dev/test-infra/tools/kntest/cmd/kntest@latest"+
				" kubetest2 gke --max-nodes=1 --machine=e2-standard-2 "+
				"--enable-workload-identity --cluster-version=latest "+
				"--extra-gcloud-flags --enable-stackdriver-kubernetes "+
				"--no-enable-ip-alias --no-enable-autoupgrade "+
				"--addons=Istio,NodeLocalDNS "+
				"--test-command=%s/test/e2e-tests.sh "+
				"--run-tests --extra-gcloud-flags= --logging=NONE "+
				"--monitoring=NONE", rootDir)),
			contains("INTEGRATION TESTS PASSED"),
		},
	}}
	for _, tc := range tcs {
		tc := tc
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
	rng1 := randString(12)
	rng2 := randString(12)
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
