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

set -Eeo pipefail

pushd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." > /dev/null
# shellcheck disable=SC1090
source "$(go run ./cmd/script e2e-tests.sh)"
popd > /dev/null

export INGRESS_CLASS=${INGRESS_CLASS:-istio.ingress.networking.knative.dev}

function is_ingress_class() {
  [[ "${INGRESS_CLASS}" == *"${1}"* ]]
}

# Copied from https://github.com/knative/client/blob/main/test/common.sh#L32
function install_istio() {
  if [[ -z "${ISTIO_VERSION:-}" ]]; then
    readonly ISTIO_VERSION="latest"
  fi

  header "Installing Istio ${ISTIO_VERSION}"
  local LATEST_NET_ISTIO_RELEASE_VERSION=$(curl -L --silent "https://api.github.com/repos/knative/net-istio/releases" | \
    jq -r '[.[].tag_name] | sort_by( sub("knative-";"") | sub("v";"") | split(".") | map(tonumber) ) | reverse[0]')
  # And checkout the setup script based on that release
  local NET_ISTIO_DIR=$(mktemp -d)
  (
    cd $NET_ISTIO_DIR \
      && git init \
      && git remote add origin https://github.com/knative-extensions/net-istio.git \
      && git fetch --depth 1 origin $LATEST_NET_ISTIO_RELEASE_VERSION \
      && git checkout FETCH_HEAD
  )

  echo "Resolved net-istio release: ${LATEST_NET_ISTIO_RELEASE_VERSION}"
  echo "net-istio commit SHA: $(git -C "${NET_ISTIO_DIR}" rev-parse HEAD)"

  if [[ -z "${ISTIO_PROFILE:-}" ]]; then
    readonly ISTIO_PROFILE="istio-ci-no-mesh"
  fi
  # Accept legacy ISTIO_PROFILE values with a trailing .yaml suffix.
  # Use a local because ISTIO_PROFILE is readonly once set above.
  local istio_profile="${ISTIO_PROFILE%.yaml}"
  local istio_dir="${NET_ISTIO_DIR}/third_party/istio-${ISTIO_VERSION}/${istio_profile}"

  if [[ ! -d "${istio_dir}" ]]; then
    echo "Istio profile directory not found: ${istio_dir}" >&2
    return 1
  fi

  if [[ -n "${CLUSTER_DOMAIN:-}" ]]; then
    find "${istio_dir}" -type f -name '*.yaml' -print0 \
      | xargs -0 sed -i.bak -e "s#cluster\.local#${CLUSTER_DOMAIN}#g"
    # Clean up the .bak files left behind by sed -i.bak so that any future
    # "kubectl apply -f <dir>" passes do not trip over them.
    find "${istio_dir}" -type f -name '*.yaml.bak' -delete
  fi

  echo ">> Installing Istio"
  echo "Istio version: ${ISTIO_VERSION}"
  echo "Istio profile: ${istio_profile}"
  kubectl apply -f "${istio_dir}" || {
    echo "Failed to apply Istio manifests from ${istio_dir}" >&2
    return 1
  }

  wait_until_pods_running istio-system || {
    echo "Istio pods in istio-system did not become ready" >&2
    return 1
  }
}

function knative_setup() {
  if is_ingress_class istio; then
    install_istio
  fi
  start_latest_knative_serving
}

# Script entry point.
initialize "$@" --cloud-provider kind -v 9

go_test_e2e ./test/e2e || fail_test

success
