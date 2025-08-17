#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
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

# **NOTE** When adding new CRDs. Make sure to include the new rbac permissions in docs/deploy/resources/rbac.yaml

set -o errexit
set -o nounset
set -o pipefail

usage() {
    echo "Usage: $0 [phase...]"
    echo ""
    echo "If no phases are specified, all phases will be run."
    echo "Available phases:"
    echo "  composite"
    echo "  backendconfig"
    echo "  frontendconfig"
    echo "  svcneg"
    echo "  negbinding"
    echo "  providerconfig"
    echo "  serviceattachment"
    echo "  help"
    exit 1
}

if [[ "$#" -gt 0 && ("$1" == "-h" || "$1" == "--help" || "$1" == "help") ]]; then
    usage
fi

# TODO: temporarily disable the other codegen.
# ALL_PHASES=(composite backendconfig frontendconfig svcneg negbinding providerconfig serviceattachment)
ALL_PHASES=(negbinding)
PHASES_TO_RUN=()

if [ "$#" -eq 0 ]; then
    PHASES_TO_RUN=("${ALL_PHASES[@]}")
else
    for arg in "$@"; do
        # check if arg is a valid phase
        found=0
        for phase in "${ALL_PHASES[@]}"; do
            if [[ "$phase" == "$arg" ]]; then
                found=1
                break
            fi
        done

        if [[ ${found} -eq 0 ]]; then
            echo "Error: Invalid phase '$arg'"
            usage
        fi
        PHASES_TO_RUN+=("$arg")
    done
fi

should_run() {
    local phase_name="$1"
    for phase in "${PHASES_TO_RUN[@]}"; do
        if [ "$phase" == "$phase_name" ]; then
            return 0
        fi
    done
    return 1
}

generate_for_api() {
  # $1: api name (e.g. backendconfig)
  # $2: client package path (e.g. pkg/backendconfig)
  # $3: versions (comma-separated) (e.g. v1beta1)
  local api_name="$1"
  local client_package="$2"
  local versions_csv="$3"

  local report_filename="hack/${api_name}.openapi-violations.txt"
  local apis_root="pkg/apis"
  local go_pkg_root="k8s.io/ingress-gce"

  echo "[API] ${api_name}, client package: ${client_package}, versions: ${versions_csv}"

  local api_packages=()
  IFS=',' read -ra versions <<< "$versions_csv"
  for version in "${versions[@]}"; do
    api_packages+=("${apis_root}/${api_name}/${version}")
  done

  for api_package in "${api_packages[@]}"; do
    echo "[GEN] helpers for ${api_package}..."
    kube::codegen::gen_helpers \
      --boilerplate "${BOILERPLATE_TXT}" \
      "${api_package}"
  done

  for api_package in "${api_packages[@]}"; do
    echo "[GEN] register for ${api_package}..."
    kube::codegen::gen_register \
      --boilerplate "${BOILERPLATE_TXT}" \
      "${api_package}"
  done

  for version in "${versions[@]}"; do
    local api_and_ver="${api_name}/${version}"

    echo "[GEN] client for ${api_and_ver}..."
    kube::codegen::gen_client \
      --boilerplate "${BOILERPLATE_TXT}" \
      --with-watch \
      --output-dir "${client_package}" \
      --output-pkg "${go_pkg_root}/${client_package}" \
      --one-input-api "${api_and_ver}" \
      "${apis_root}"
  done

if [[ "${UPDATE_API_KNOWN_VIOLATIONS:-}" == "true" ]]; then
    update_report="--update-report"
fi

for api_package in "${api_packages[@]}"; do
    echo "[GEN] openapi for ${api_package}..."
    kube::codegen::gen_openapi \
      --boilerplate "${BOILERPLATE_TXT}" \
      --output-dir "${api_package}" \
      --output-pkg "k8s.io/ingress-gce/${api_package}" \
      --report-filename "${report_filename:-/dev/null}" \
      ${update_report:+"${update_report}"} \
      "${api_package}"
  done
}


ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SCRIPT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BOILERPLATE_TXT="${ROOT}/hack/boilerplate.go.txt"

export GOBIN="${SCRIPT_ROOT}/tools/bin"
export PATH="${GOBIN}:${PATH}"
GOPATH="$(go env GOPATH)"
export GOPATH

echo "Using following variables for code generation:"
echo ""
echo "ROOT=${ROOT}"
echo "SCRIPT_ROOT=${SCRIPT_ROOT}"
echo "GOPATH=${GOPATH}"
echo ""

CODEGEN_PKG=$(go env GOMODCACHE)/k8s.io/code-generator@v0.31.12
CODEGEN_SCRIPT="${CODEGEN_PKG}/kube_codegen.sh"
echo "Using codegen script ${CODEGEN_SCRIPT}"
# shellcheck disable=SC1090
source "${CODEGEN_SCRIPT}"

echo
echo

if should_run "composite"; then
  echo "[GEN] composite types"
  pushd "${ROOT}"
  go run "pkg/composite/gen/main.go"
  popd
fi

if should_run "backendconfig"; then
  generate_for_api "backendconfig" "pkg/backendconfig/client" "v1beta1"
fi

if should_run "frontendconfig"; then
  generate_for_api "frontendconfig" "pkg/frontendconfig/client" "v1beta1"
fi

if should_run "svcneg"; then
  generate_for_api "svcneg" "pkg/svcneg/client" "v1beta1"
fi

if should_run "negbinding"; then
  generate_for_api "negbinding" "pkg/negbinding" "v1"
fi

if should_run "providerconfig"; then
  generate_for_api "providerconfig" "pkg/providerconfig/client" "v1"
fi

if should_run "serviceattachment"; then
  generate_for_api "serviceattachment" "pkg/serviceattachment/client" "v1beta1,v1"
fi
