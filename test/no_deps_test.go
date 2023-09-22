package sample_test

import (
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestNoDependenciesPresent(t *testing.T) {
	var (
		bytes []byte
		err   error
	)
	if bytes, err = os.ReadFile("../go.mod"); err != nil {
		t.Fatal(err)
	}

	var mf *modfile.File
	if mf, err = modfile.ParseLax("go.mod", bytes, nil); err != nil {
		t.Fatal(err)
	}

	if len(mf.Require) > 0 {
		deps := make([]string, 0, len(mf.Require))
		for _, r := range mf.Require {
			deps = append(deps, r.Mod.String())
		}
		t.Errorf("go.mod file should not have dependencies, but has %d: %+q",
			len(mf.Require), deps)
	}
}
