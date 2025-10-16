package unit_test

import (
	"fmt"
	"go/build"
	"os"
	"path"
	"strings"
	"testing"

	"knative.dev/hack/pkg/utest/require"
)

func TestCodegen(t *testing.T) {
	t.Parallel()
	wantRetcode := 248
	sc := newShellScript(
		mockDeepcopyGen(t, func(gen *deepcopyGen) {
			gen.retcode = wantRetcode
		}),
		fakeProwJob(),
		loadFile("source-codegen-library.bash"),
		mockGo(),
	)
	tcs := []testCase{{
		name: "generate-groups deepcopy " +
			"knative.dev/hack/test/codegen/testdata/apis/hack/v1alpha1 " +
			"hack:v1alpha1",
		retcode: retcode(wantRetcode),
		stdout: equal(`WARNING: generate-groups.sh is deprecated.
WARNING: Please use k8s.io/code-generator/kube_codegen.sh instead.

WARNING: generate-internal-groups.sh is deprecated.
WARNING: Please use k8s.io/code-generator/kube_codegen.sh instead.

👻 go install k8s.io/code-generator/cmd/applyconfiguration-gen k8s.io/code-generator/cmd/client-gen k8s.io/code-generator/cmd/conversion-gen k8s.io/code-generator/cmd/deepcopy-gen k8s.io/code-generator/cmd/defaulter-gen k8s.io/code-generator/cmd/informer-gen k8s.io/code-generator/cmd/lister-gen k8s.io/code-generator/cmd/openapi-gen
Generating deepcopy funcs
Deepcopy generation complete
--- Cleaning up generated code
`,
		),
		stderr: warned("Failed to determine the knative.dev/pkg package"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

type deepcopyGen struct {
	retcode int
}

type deepcopyGenOpt func(*deepcopyGen)

func mockDeepcopyGen(t TestingT, opts ...deepcopyGenOpt) scriptlet {
	d := deepcopyGen{}
	for _, opt := range opts {
		opt(&d)
	}
	execPermission := os.FileMode(0o755)
	slet := instructions()
	gobin := path.Join(currentGopath(), "bin")
	deepcopyGenPath := path.Join(gobin, "deepcopy-gen")
	if !isCheckoutOntoGopath() {
		tmpdir := t.TempDir()
		slet = envs(map[string]string{"TMPDIR": tmpdir})
		gobin = path.Join(tmpdir, "go", "bin")
		deepcopyGenPath = path.Join(gobin, "deepcopy-gen")
		require.NoError(t, os.MkdirAll(gobin, execPermission))
	} else {
		t.Cleanup(func() {
			require.NoError(t, os.Remove(deepcopyGenPath))
		})
	}
	require.NoError(t, os.WriteFile(
		deepcopyGenPath,
		// restore zz_generated files, if they are deleted
		[]byte(strings.Join([]string{"#!/bin/bash",
			`set -Eeuo pipefail`,
			`git diff --numstat | grep -E '0\s+[0-9]{2,}.+zz_generated.+\.go' ` +
				`| awk '{print $3}' | xargs -r git restore`,
			`echo "Deepcopy generation complete"`,
			fmt.Sprint("exit ", d.retcode)}, "\n")),
		execPermission))
	return slet
}

// isCheckoutOntoGopath checks if the current directory is under GOPATH
func isCheckoutOntoGopath() bool {
	rootDir := path.Dir(path.Dir(currentDir()))
	return strings.HasPrefix(rootDir, currentGopath())
}

func currentGopath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}
