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

package main_test

import (
	"bytes"
	"testing"

	main "knative.dev/hack/cmd/script"
	"knative.dev/hack/pkg/inflator/cli"
	"knative.dev/hack/pkg/utest/assert"
)

func TestMainFn(t *testing.T) {
	var buf bytes.Buffer
	var retcode = -1_234_567_890 // nolint:gomnd // gate value
	withOptions(
		func() {
			main.RunMain()
		},
		func(ex *cli.Execution) {
			ex.Stdout = &buf
			ex.Stderr = &buf
			ex.Args = []string{"--help"}
			ex.Exit = func(c int) {
				retcode = c
			}
		},
	)
	assert.Equal(t, 0, retcode)
	assert.ContainsSubstring(t, buf.String(), "Hacks as Go self-extracting binary")
}

func withOptions(fn func(), options ...cli.Option) {
	prev := cli.Options
	cli.Options = options
	defer func(p []cli.Option) {
		cli.Options = p
	}(prev)
	fn()
}
