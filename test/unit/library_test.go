package unit_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelperFunctions(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile(
		"source-library.bash",
		"fake-prow-job.bash",
	), mockGo(), mockKubectl(map[string]string{
		"get pods -n test-infra --selector=app=controller": "acme\nexample\nknative",
	}))
	tcs := []testCase{{
		name:   `echo "$REPO_NAME_FORMATTED"`,
		stdout: lines("Knative Hack"),
	}, {
		name: `get_canonical_path test/unit/library_test.go`,
		stdout: []check{func(t assert.TestingT, output string) {
			assert.Contains(t, output, "hack/test/unit/library_test.go")
			pth := strings.Trim(output, "\n")
			fi, err := os.Stat(pth)
			assert.NoError(t, err)
			assert.False(t, fi.IsDir())
			assert.True(t, path.IsAbs(pth))
		}},
	}, {
		name:   `capitalize "foo bar"`,
		stdout: lines("Foo Bar"),
	}, {
		name: `dump_app_logs "controller" "test-infra"`,
		stdout: lines(
			">>> Knative Hack controller logs:",
			">>> Pod: acme",
			"kubectl -n test-infra logs acme --all-containers",
			">>> Pod: example",
			"kubectl -n test-infra logs example --all-containers",
			">>> Pod: knative",
			"kubectl -n test-infra logs knative --all-containers",
		),
	}, {
		name:   `is_protected_gcr "gcr.io/knative-releases"`,
		stdout: equal(""),
	}, {
		name:   `is_protected_gcr "gcr.io/knative-nightly"`,
		stdout: equal(""),
	}, {
		name:    `is_protected_gcr "gcr.io/knative-foobar"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_gcr "gcr.io/foobar-releases"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_gcr "gcr.io/foobar-nightly"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_gcr ""`,
		retcode: retcode(1),
	}, {
		name: `is_protected_cluster "gke_knative-tests_us-central1-f_prow"`,
	}, {
		name: `is_protected_cluster "gke_knative-tests_us-west2-a_prow"`,
	}, {
		name: `is_protected_cluster "gke_knative-tests_us-west2-a_foobar"`,
	}, {
		name:    `is_protected_cluster "gke_knative-foobar_us-west2-a_prow"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_cluster ""`,
		retcode: retcode(1),
	}, {
		name: `is_protected_project "knative-tests"`,
	}, {
		name:    `is_protected_project "knative-foobar"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_project "foobar-tests"`,
		retcode: retcode(1),
	}, {
		name:    `is_protected_project ""`,
		retcode: retcode(1),
	}, {
		name:   `calcRetcode "An example message"`,
		stdout: lines("254"),
	}, {
		name:   `calcRetcode ""`,
		stdout: lines("1"),
	}, {
		name:   `hashCode "An example message"`,
		stdout: equal("623783294"),
	}, {
		name:   `hashCode ""`,
		stdout: equal("0"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
