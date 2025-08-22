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

readonly ZK_JAR_NAME="zookeeper-3.4.9-fatjar.jar"
readonly REMOTE_JAR_URL="https://github.com/dubbogo/resources/raw/master/zookeeper-4unitest/contrib/fatjar/${ZK_JAR_NAME}"
readonly ZK_JAR_PATH="pkg/registry/zookeeper-4unittest/contrib/fatjar"
readonly TARGET_JAR="${ZK_JAR_PATH}/${ZK_JAR_NAME}"

main() {
    # if already exists, skip download
    if [[ -f "${TARGET_JAR}" ]]; then
        echo "Zookeeper exists at ${TARGET_JAR}, skipping download."
        return 0
    fi

    echo "--> Downloading zookeeper binary..."

    mkdir -p "${ZK_JAR_PATH}"

    if command -v curl >/dev/null 2>&1; then
        curl -fL -o "${TARGET_JAR}" "${REMOTE_JAR_URL}"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "${TARGET_JAR}" "${REMOTE_JAR_URL}"
    else
        echo "Error: Neither curl nor wget is available. Cannot download dependencies." >&2
        exit 1
    fi

    echo "Zookeeper downloaded successfully to ${TARGET_JAR}."
}

main