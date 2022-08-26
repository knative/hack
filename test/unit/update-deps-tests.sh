#!/usr/bin/env bash

# Copyright 2022 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

source "$(dirname "${BASH_SOURCE[0]:-$0}")/test-helper.sh"
source "$(dirname "${BASH_SOURCE[0]:-$0}")/../../library.sh"

set -Eeuo pipefail

function mock_go_update_deps() {
  function go() {
	  echo "go $*"
  }
  go_update_deps "$@" 2>&1
}

test_function "${FAILURE}" 'unknown option --unknown' go_update_deps --unknown
test_function "${SUCCESS}" 'Update Deps' mock_go_update_deps
test_function "${SUCCESS}" 'Golang module: knative.dev/hack/test/e2e' mock_go_update_deps
test_function "${SUCCESS}" 'Golang module: knative.dev/hack/schema' mock_go_update_deps
test_function "${SUCCESS}" 'Golang module: knative.dev/hack' mock_go_update_deps
test_function "${SUCCESS}" 'Updating licenses' mock_go_update_deps
test_function "${SUCCESS}" 'Removing unwanted vendor files' mock_go_update_deps
test_function "${SUCCESS}" 'go mod tidy' mock_go_update_deps
test_function "${SUCCESS}" 'go mod vendor' mock_go_update_deps
test_function "${SUCCESS}" 'go run github.com/google/go-licenses@v1.2.1 save ./... --save_path=third_party/VENDOR-LICENSE --force' mock_go_update_deps
test_function "${SUCCESS}" 'go run knative.dev/test-infra/buoy@latest float ./go.mod --release v9000.1 --domain knative.dev' mock_go_update_deps --upgrade
test_function "${SUCCESS}" 'go run knative.dev/test-infra/buoy@latest float ./go.mod --release 1.25 --domain knative.dev --module-release 0.28' mock_go_update_deps --upgrade --release 1.25 --module-release 0.28
