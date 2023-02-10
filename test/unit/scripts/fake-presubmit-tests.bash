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

add_trap teardown_fake_presubmit EXIT

VALIDATION_TESTS="$(mktemp -d)/presubmit-tests.sh"

cat <<EOF > "${VALIDATION_TESTS}"
#!/usr/bin/env bash

echo ">> Running fake presubmit tests"
echo "UNIT TESTS PASSED"
echo "INTEGRATION TESTS PASSED"
EOF

chmod +x "${VALIDATION_TESTS}"

function teardown_fake_presubmit() {
  rm -rf "${VALIDATION_TESTS}"
}
