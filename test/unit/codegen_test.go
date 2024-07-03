package unit_test

import (
	"os"
	"strings"
	"testing"

	"knative.dev/hack/pkg/utest/require"
)

func TestCodegen(t *testing.T) {
	t.Parallel()
	tmpdir := t.TempDir()
	execPermission := os.FileMode(0o755)
	require.NoError(t, os.MkdirAll(tmpdir+"/go/bin", execPermission))
	require.NoError(t, os.WriteFile(
		tmpdir+"/go/bin/deepcopy-gen",
		[]byte(strings.Join([]string{"#!/bin/bash",
			`git restore test/e2e/apis/hack/v1alpha1/zz_generated.deepcopy.go`,
			`echo "Deepcopy generation complete"`,
			"exit 248"}, "\n")),
		execPermission))
	sc := newShellScript(
		envs(map[string]string{"TMPDIR": tmpdir}),
		loadFile("source-codegen-library.bash"),
		mockGo(),
	)
	tcs := []testCase{{
		name: "generate-groups deepcopy " +
			"knative.dev/hack/test/e2e/apis/hack/v1alpha1/generated " +
			"knative.dev/hack/test/e2e/apis " +
			"hack:v1alpha1",
		retcode: retcode(248),
		stdout: equal(`WARNING: generate-groups.sh is deprecated.
WARNING: Please use k8s.io/code-generator/kube_codegen.sh instead.

WARNING: generate-internal-groups.sh is deprecated.
WARNING: Please use k8s.io/code-generator/kube_codegen.sh instead.

go install k8s.io/code-generator/cmd/applyconfiguration-gen k8s.io/code-generator/cmd/client-gen k8s.io/code-generator/cmd/conversion-gen k8s.io/code-generator/cmd/deepcopy-gen k8s.io/code-generator/cmd/defaulter-gen k8s.io/code-generator/cmd/informer-gen k8s.io/code-generator/cmd/lister-gen k8s.io/code-generator/cmd/openapi-gen
Generating deepcopy funcs
Deepcopy generation complete
`,
		),
		stderr: equal(strings.TrimLeft(`
╭───────────────────────────────────────────────────────────╮
│                                                           │
│   WARN: Failed to determine the knative.dev/pkg package   │
│                                                           │
╰───────────────────────────────────────────────────────────╯
--- Cleaning up generated code
`, "\n")),
	}}
	for i := range tcs {
		tc := tcs[i]
		t.Run(tc.name, tc.test(sc))
	}
}
