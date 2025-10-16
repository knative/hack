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

rootdir="$(dirname "${BASH_SOURCE[0]:-$0}")/../.."
relative_rootdir="$(realpath -s --relative-to="$PWD" "$rootdir")"
# shellcheck disable=SC1090
source "$(go run "${relative_rootdir}/cmd/script" codegen-library.sh)"

generate-groups deepcopy \
  knative.dev/hack/test/codegen/testdata/apis/hack/v1alpha1 \
  knative.dev/hack/test/codegen/testdata/apis \
  hack:v1alpha1 \
  "$@"
