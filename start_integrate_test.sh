#!/bin/bash

#
#  Licensed to the Apache Software Foundation (ASF) under one or more
#  contributor license agreements.  See the NOTICE file distributed with
#  this work for additional information regarding copyright ownership.
#  The ASF licenses this file to You under the Apache License, Version 2.0
#  (the "License"); you may not use this file except in compliance with
#  the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -euo pipefail

readonly SAMPLES_REPO="https://github.com/apache/dubbo-go-pixiu-samples.git"
readonly SAMPLES_DIR="integrate_samples"

readonly HEAD_REPO_SLUG="${1-}" # e.g., "my-fork/dubbo-go-pixiu"
readonly HEAD_COMMIT_SHA="${2-}" # The specific commit SHA to test
readonly BASE_BRANCH="${3-}"    # The target branch of the PR, e.g., "main"

main() {
    if [[ -z "${HEAD_REPO_SLUG}" || -z "${HEAD_COMMIT_SHA}" || -z "${BASE_BRANCH}" ]]; then
        echo "Error: Missing required arguments." >&2
        echo "Usage: $0 <head-repo-slug> <head-commit-sha> <base-branch>" >&2
        exit 1
    fi

    echo "Starting integration test setup..."
    echo "--------------------------------------------------"
    echo "  Testing Commit:   ${HEAD_COMMIT_SHA}"
    echo "  From Repository:  ${HEAD_REPO_SLUG}"
    echo "  Against Branch:   ${BASE_BRANCH}"
    echo "--------------------------------------------------"

    rm -rf "${SAMPLES_DIR}"

    echo "--> Cloning samples repository from branch '${BASE_BRANCH}'..."
    git clone -b "${BASE_BRANCH}" --depth 1 "${SAMPLES_REPO}" "${SAMPLES_DIR}"
    cd "${SAMPLES_DIR}"
    echo "--> Successfully cloned and entered '${SAMPLES_DIR}'."

    local module_to_replace="github.com/apache/dubbo-go-pixiu"
    local replacement_path="github.com/${HEAD_REPO_SLUG}@${HEAD_COMMIT_SHA}"
    echo "--> Replacing module '${module_to_replace}' with '${replacement_path}'..."
    go mod edit -replace="${module_to_replace}=${replacement_path}"

    echo "--> Tidying Go modules to fetch the new dependency..."
    go mod tidy

    echo "--> Handing off to the test runner inside the samples repository..."
    ./start_integrate_test.sh
}

main "$@"