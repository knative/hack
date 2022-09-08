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

	"github.com/stretchr/testify/assert"
	"github.com/wavesoftware/go-commandline"
	main "knative.dev/hack/cmd/script"
	"knative.dev/hack/pkg/inflator/cli"
)

func TestMainFn(t *testing.T) {
	var buf bytes.Buffer
	var retcode *int
	withOptions(
		func() {
			main.RunMain()
		},
		commandline.WithArgs("--help"),
		commandline.WithOutput(&buf),
		commandline.WithExit(func(c int) {
			retcode = &c
		}),
	)
	assert.Nil(t, retcode)
	assert.Contains(t, buf.String(), "Script will extract Hack scripts")
}

func withOptions(fn func(), options ...commandline.Option) {
	prev := cli.Options
	cli.Options = options
	defer func(p []commandline.Option) {
		cli.Options = p
	}(prev)
	fn()
}
