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

# Fake we're in a Prow job, if running locally.
if [[ -z "${PROW_JOB_ID:-}" ]]; then
  export PROW_JOB_ID=123
  export JOB_TYPE='presubmit'
  export PULL_PULL_SHA='deadbeef1234567890'
fi
export KNATIVE_HACK_SCRIPT_MANUAL_VERBOSE=true
