#!/usr/bin/env bash

# Copyright 2024 The Knative Authors
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

set -Eeuo pipefail

rootdir="$(realpath "$(dirname "${BASH_SOURCE[0]:-$0}")/../..")"
cd "${rootdir}"

# shellcheck disable=SC1090
source "$(go run ./cmd/script library.sh)"

./test/hack/update-codegen.sh

if ! git diff --exit-code; then
  abort "codegen is out of date, please run test/hack/update-codegen.sh, and commit the changes."
fi

header "Codegen is up to date"
