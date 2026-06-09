#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Integration test runner for the void-glibc image.
# Covers: void-glibc, void-musl
#
# Usage: void-glibc.sh <backend> <mode>
#   backend: podman | docker | nerdctl
#   mode:    rootless | rootful

set -e

BACKEND="${1:?backend argument required (podman|docker|nerdctl)}"
MODE="${2:?mode argument required (rootless|rootful)}"
IMAGE="void-glibc"
CONTAINER_BASIC="otter-test-void"
CONTAINER_FULL="otter-test-void-full"
REPORT_FILE="test-report-void-glibc-${BACKEND}-${MODE}.txt"
FAILURES=0

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

ROOT_FLAG=""
if [ "${MODE}" = "rootful" ]; then
    ROOT_FLAG="--root"
fi

OTTER="otter --container-manager ${BACKEND} ${ROOT_FLAG}"

pass() {
    printf "[PASS] %s\n" "$1" | tee -a "${REPORT_FILE}"
}

fail() {
    printf "[FAIL] %s\n  => %s\n" "$1" "$2" | tee -a "${REPORT_FILE}"
    FAILURES=$((FAILURES + 1))
}

run() {
    label="$1"
    shift
    if output=$(sh -c "$*" 2>&1); then
        pass "${label}"
    else
        fail "${label}" "${output}"
    fi
}

# ----------------------------------------------------------------------------
# Header
# ----------------------------------------------------------------------------

printf "=== Integration Test: void-glibc | backend=%s | mode=%s ===\n" \
    "${BACKEND}" "${MODE}" | tee "${REPORT_FILE}"
printf "Date: %s\n\n" "$(date -u)" | tee -a "${REPORT_FILE}"

# ----------------------------------------------------------------------------
# Basic lifecycle
# ----------------------------------------------------------------------------

printf "\n--- Basic lifecycle ---\n" | tee -a "${REPORT_FILE}"

run "create basic" \
    "${OTTER} create ${CONTAINER_BASIC} --image ${IMAGE}"

run "start basic" \
    "${OTTER} start ${CONTAINER_BASIC}"

run "enter basic: echo hello" \
    "${OTTER} enter ${CONTAINER_BASIC} --no-tty -- echo hello"

run "stop basic" \
    "${OTTER} stop ${CONTAINER_BASIC}"

run "upgrade basic" \
    "${OTTER} upgrade ${CONTAINER_BASIC}"

run "remove basic" \
    "${OTTER} remove ${CONTAINER_BASIC} --force"

# ----------------------------------------------------------------------------
# Full-featured lifecycle
# ----------------------------------------------------------------------------

printf "\n--- Full-featured lifecycle ---\n" | tee -a "${REPORT_FILE}"

run "create full" \
    "${OTTER} create ${CONTAINER_FULL} \
        --image ${IMAGE} \
        --shell bash \
        --hostname otter-test-host \
        --memory 512m \
        --cpu-threads 2 \
        --volume /tmp:/tmp/otter-host \
        --unshare-netns \
        --unshare-ipc \
        --unshare-process"

run "start full" \
    "${OTTER} start ${CONTAINER_FULL}"

run "enter full: uname -a" \
    "${OTTER} enter ${CONTAINER_FULL} --no-tty --no-workdir --clean-path --empty-env -- uname -a"

run "stop full" \
    "${OTTER} stop ${CONTAINER_FULL}"

run "upgrade full: --all --running" \
    "${OTTER} upgrade --all --running"

run "remove full" \
    "${OTTER} remove ${CONTAINER_FULL} --force --rm-home --bypass-lock"

# ----------------------------------------------------------------------------
# Report
# ----------------------------------------------------------------------------

printf "\n=== Results: void-glibc | backend=%s | mode=%s ===\n" \
    "${BACKEND}" "${MODE}" | tee -a "${REPORT_FILE}"

if [ "${FAILURES}" -eq 0 ]; then
    printf "All tests passed.\n" | tee -a "${REPORT_FILE}"
    exit 0
else
    printf "%d test(s) failed. See above for details.\n" "${FAILURES}" | tee -a "${REPORT_FILE}"
    exit 1
fi