#!/usr/bin/env bash

# Copyright 2018 The Knative Authors
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

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

set -Eeuo pipefail

pushd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." > /dev/null
# shellcheck disable=SC1090
source "$(go run ./cmd/script presubmit-tests.sh)"
popd > /dev/null

# Run our custom build tests after the standard build tests.

function post_build_tests() {
  local failed=0
  for script in *.sh; do
    subheader "Checking integrity of ${script}"
    bash -c "source ${script}" || { failed=1; echo "--- FAIL: ${script}"; }
  done
  return ${failed}
}

# We use the default integration test runner.

main "$@"
