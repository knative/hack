/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hack_test

import (
	"testing"

	"knative.dev/hack"
	"knative.dev/hack/pkg/utest/assert"
	"knative.dev/hack/pkg/utest/require"
)

func TestScriptsAreEmbedded(t *testing.T) {
	entries, err := hack.Scripts.ReadDir(".")
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
		if !entry.IsDir() {
			fi, err := entry.Info()
			require.NoError(t, err)
			assert.Greater(t, fi.Size(), int64(0))
		}
	}
	requiredLibs := []string{
		"codegen-library.sh",
		"e2e-tests.sh",
		"infra-library.sh",
		"library.sh",
		"microbenchmarks.sh",
		"performance-tests.sh",
		"presubmit-tests.sh",
		"release.sh",
		"shellcheck-presubmit.sh",
	}
	for _, lib := range requiredLibs {
		assert.Contains(t, names, lib)
	}
}
