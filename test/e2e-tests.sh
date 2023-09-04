#!/usr/bin/env bash

# Copyright 2019 The Knative Authors
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

# This script runs the end-to-end tests.

# If you already have a Knative cluster setup and kubectl pointing
# to it, call this script with the --run-tests arguments and it will use
# the cluster and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, run the tests and delete the cluster.

set -Eeuo pipefail

pushd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." > /dev/null
# shellcheck disable=SC1090
source "$(go run ./cmd/script e2e-tests.sh)"
popd > /dev/null

function knative_setup() {
  start_latest_knative_serving
  export KNATIVE_SETUP_DONE=1
}

function test_setup() {
  export TEST_SETUP_DONE=1
}

function dump_metrics() {
  header ">> Starting kube proxy"
  header ">> Grabbing k8s metrics"
}

# Script entry point.
initialize "$@" --num-nodes=1 --machine-type=e2-standard-4 \
  --enable-workload-identity --cluster-version=latest \
  --gcloud-extra-flags "--logging=NONE --monitoring=NONE"

[[ ${KNATIVE_SETUP_DONE:-0} == 1 ]] || fail_test 'Knative setup not persisted'
[[ ${TEST_SETUP_DONE:-0} == 1 ]] || fail_test 'Test setup not persisted'

go_test_e2e ./test/e2e

success
