#!/usr/bin/env bash

# Copyright 2020 The Knative Authors
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

# Setup the env for doing Knative style codegen.

# Store Bash options
oldstate="$(set +o)"

set -Eeuo pipefail

export kn_hack_dir kn_hack_library \
  MODULE_NAME GOPATH GOBIN \
  CODEGEN_PKG KNATIVE_CODEGEN_PKG

kn_hack_dir="$(realpath "$(dirname "${BASH_SOURCE[0]:-$0}")")"
kn_hack_library=${kn_hack_library:-"${kn_hack_dir}/library.sh"}

if [[ -f "$kn_hack_library" ]]; then
  # shellcheck disable=SC1090
  source "$kn_hack_library"
else
  echo "The \$kn_hack_library points to a non-existent file: $kn_hack_library" >&2
  exit 42
fi

# Change dir to the original executing script's directory, not the current source!
pushd "$(dirname "$(realpath "$0")")" > /dev/null

function go-resolve-pkg-dir() {
  local pkg="${1:?Pass the package name}"
  local repodir pkgdir
  repodir="$(go_run knative.dev/toolbox/modscope@latest current --path)"
  if [ -d "${repodir}/vendor" ]; then
    pkgdir="${repodir}/vendor/${pkg}"
    if [ -d "${pkgdir}" ]; then
      echo "${pkgdir}"
      return 0
    else
      return 1
    fi
  else
    go list -f '{{.Dir}}' "${pkg}" 2>/dev/null
    return $?
  fi
}

if ! CODEGEN_PKG="${CODEGEN_PKG:-"$(go-resolve-pkg-dir k8s.io/code-generator)"}"; then
  warning "Failed to determine the k8s.io/code-generator package"
fi
if ! KNATIVE_CODEGEN_PKG="${KNATIVE_CODEGEN_PKG:-"$(go-resolve-pkg-dir knative.dev/pkg)"}"; then
  warning "Failed to determine the knative.dev/pkg package"
fi

popd > /dev/null

MODULE_NAME=$(go_mod_module_name)
GOPATH=$(go_mod_gopath_hack)
GOBIN="${TMPDIR}/${MODULE_NAME}/bin" # Set GOBIN explicitly as k8s-gen' are installed by go install.

if [[ -n "${CODEGEN_PKG}" ]] && ! [ -x "${CODEGEN_PKG}/generate-groups.sh" ]; then
  chmod +x "${CODEGEN_PKG}/generate-groups.sh"
  chmod +x "${CODEGEN_PKG}/generate-internal-groups.sh"
fi
if [[ -n "${KNATIVE_CODEGEN_PKG}" ]] && ! [ -x "${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" ]; then
  chmod +x "${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh"
fi

# Generate boilerplate file with the current year
function boilerplate() {
  local go_header_file="${kn_hack_dir}/boilerplate.go.txt"
  local current_boilerplate_file="${TMPDIR}/boilerplate.go.txt"
  # Replace #{YEAR} with the current year
  sed "s/#{YEAR}/$(date +%Y)/" \
    < "${go_header_file}" \
    > "${current_boilerplate_file}"
  echo "${current_boilerplate_file}"
}

# Generate K8s' groups codegen
function generate-groups() {
  if [[ -z "${CODEGEN_PKG}" ]]; then
    abort "CODEGEN_PKG is not set"
  fi
  "${CODEGEN_PKG}"/generate-groups.sh \
    "$@" \
    --go-header-file "$(boilerplate)"
}

# Generate K8s' internal groups codegen
function generate-internal-groups() {
  if [[ -z "${CODEGEN_PKG}" ]]; then
    abort "CODEGEN_PKG is not set"
  fi
  "${CODEGEN_PKG}"/generate-internal-groups.sh \
    "$@" \
    --go-header-file "$(boilerplate)"
}

# Generate Knative style codegen
function generate-knative() {
  if [[ -z "${KNATIVE_CODEGEN_PKG}" ]]; then
    abort "KNATIVE_CODEGEN_PKG is not set"
  fi
  "${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" \
    "$@" \
    --go-header-file "$(boilerplate)"
}

# Cleanup generated code if it differs only in the boilerplate year
function cleanup-codegen() {
  log "Cleaning up generated code"
  # list git changes and skip those which differ only in the boilerplate year
  while read -r file; do
    # check if the file contains just the change in the boilerplate year
    if [[ "$(LANG=C git diff --exit-code --shortstat -- "$file")" == ' 1 file changed, 1 insertions(+), 1 deletions(-)' ]] && \
      [[ "$(git diff --exit-code -U1 -- "$file" | grep -Ec '^[+-]Copyright \d{4} The Knative Authors')" -eq 2 ]]; then
      # restore changes to that file
      git checkout -- "$file"
    fi
  done < <(git diff --exit-code --name-only)
}

add_trap cleanup-codegen EXIT

# Restore Bash options
eval "$oldstate"
