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

# -e: exit on error
# -u: raise error on undefined variables
# -o pipefail: any failed command in a pipeline causes the entire pipeline to fail
set -euo pipefail

run_make() {
    export P_DIR PIXIU_DIR PROJECT_NAME
    export BASE_DIR="$P_DIR/dist"

    # use "$@" to pass all arguments to make
    make -f "${PIXIU_DIR}/igt/Makefile" "$@"
}

main() {
    if [[ -z "${1-}" || ! -d "$1" ]]; then
      echo "error: need a valid test directory path as the first argument."
      echo "usage: ./integrate_test.sh <path_to_test_directory>"
      exit 1
    fi

    local P_DIR
    P_DIR=$(readlink -f "$1")
    local PIXIU_DIR
    PIXIU_DIR=$(readlink -f "$(dirname "$0")")
    local PROJECT_NAME
    PROJECT_NAME=$(basename "$P_DIR")

    # make sure cleanup runs on script exit, regardless of success or failure
    trap cleanup EXIT

    echo "--> Prepare environment (Docker Up)..."
    run_make docker-up

    echo "--> Health check..."
    run_make docker-health-check

    echo "--> Start pixiu..."
    run_make start

    echo "--> Build Pixiu..."
    run_make buildPixiu

    echo "--> Run integrate test..."
    run_make integration

    echo "Integrate succeed"
}

cleanup() {
    # $? saves the exit code of the last executed command
    local exit_code=$?
    echo "--- Running cleanup (Docker Down & Clean) ---"

    run_make clean || true
    run_make docker-down || true

    if [[ $exit_code -ne 0 ]]; then
        echo "❌ Integrate test failed with exit code: $exit_code"
    fi

    exit "$exit_code"
}

main "$@"