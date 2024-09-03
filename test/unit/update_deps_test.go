package unit_test

import (
	"testing"
)

func TestUpdateDeps(t *testing.T) {
	t.Parallel()
	sc := newShellScript(
		loadFile("source-library.bash"),
		mockGo(),
		mockBinary("truncate", response{
			args:     startsWith{"--size 0"},
			response: simply(""),
		}),
	)
	tcs := []testCase{{
		name:    "go_update_deps --unknown",
		retcode: retcode(232),
		stdout:  []check{contains("=== Update Deps for Golang module: knative.dev/hack")},
		stderr:  []check{contains("unknown option --unknown")},
	}, {
		name: "go_update_deps",
		stdout: []check{
			contains("Update Deps"),
			contains("Golang module: knative.dev/hack/test"),
			contains("Golang module: knative.dev/hack/schema"),
			contains("Golang module: knative.dev/hack"),
			contains("Checking licenses"),
			contains("Removing unwanted vendor files"),
			contains("👻 go mod tidy"),
			contains("👻 go run github.com/google/go-licenses@v1.0.0 check"),
			contains("👻 go mod download -x"),
		},
	}, {
		name: "go_update_deps --upgrade",
		stdout: []check{
			contains("👻 go run knative.dev/toolbox/buoy@latest float " +
				"./go.mod --release v9000.1 --domain knative.dev"),
		},
	}, {
		name: "go_update_deps --upgrade --release 1.25 --module-release 0.28",
		stdout: []check{
			contains("👻 go run knative.dev/toolbox/buoy@latest float " +
				"./go.mod --release 1.25 --domain knative.dev " +
				"--module-release 0.28"),
		},
	}}
	for i := range tcs {
		tc := tcs[i]
		t.Run(tc.name, tc.test(sc))
	}
}
