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

readonly REMOTE_BASE_URL="https://github.com/dubbogo/resources/raw/master/tools/license"
readonly CHECKER_NAME="license-header-checker"
readonly LICENSE_NAME="license.txt"
readonly TARGET_DIR="/tmp/tools/license"

main() {
    echo "Preparing license validation tools..."

    mkdir -p "${TARGET_DIR}"

    echo "--> Downloading license checker binary..."
    download_file "${REMOTE_BASE_URL}/${CHECKER_NAME}" "${TARGET_DIR}/${CHECKER_NAME}"

    echo "--> Downloading license template file..."
    download_file "${REMOTE_BASE_URL}/${LICENSE_NAME}" "${TARGET_DIR}/${LICENSE_NAME}"

    echo "--> Setting execute permissions for the checker..."
    chmod +x "${TARGET_DIR}/${CHECKER_NAME}"

    echo "Setup complete. Tools are ready in ${TARGET_DIR}"
}

download_file() {
    local url="$1"
    local destination="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fL -o "${destination}" "${url}"
    elif command -v wget >/dev/null 2>&1;
        wget -O "${destination}" "${url}"
    else
        echo "Error: Neither curl nor wget is available. Cannot download dependencies." >&2
        exit 1
    fi
}

main