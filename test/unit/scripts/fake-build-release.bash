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

set -Eeuo pipefail

build_dir="$(mktemp -d)"
export ARTIFACTS_TO_PUBLISH

function build_release() {
  local artifact_names
  declare -a artifact_names
  artifact_names=(
    foo-linux-amd64
    foo-linux-arm64
    foo-linux-ppc64le
    foo-linux-s390x
    foo-darwin-amd64
    foo-darwin-arm64
    foo-windows-amd64.exe
    foo.yaml
  )
  for artifact_name in "${artifact_names[@]}"; do
    uuidgen > "${build_dir}/${artifact_name}"
    echo "${build_dir}/${artifact_name}" >> "${build_dir}/artifacts.list"
  done
  ARTIFACTS_TO_PUBLISH="$(tr '\r\n' ' ' < "${build_dir}/artifacts.list")"
  if [[ -n "${CALCULATE_CHECKSUMS:-}" ]]; then
    calculate_checksums
  fi
}

function calculate_checksums {
  local checksums file
  checksums="${build_dir}/checksums.txt"
  rm -vf "${checksums}"
  while read -r file; do
    pushd "$(dirname "$file")" >/dev/null
    sha256sum "$(basename "$file")" >> "${checksums}"
    popd >/dev/null
  done < "${build_dir}/artifacts.list"
  ARTIFACTS_TO_PUBLISH="${ARTIFACTS_TO_PUBLISH} ${checksums}"
}
