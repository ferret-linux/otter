#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Integration test runner for ubuntu-specific extras:
# lock, unlock, pause, restart.
#
# Usage: ubuntu-extra.sh <backend> <mode>
#   backend: podman | docker | nerdctl
#   mode:    rootless | rootful

set -e

BACKEND="${1:?backend argument required (podman|docker|nerdctl)}"
MODE="${2:?mode argument required (rootless|rootful)}"
IMAGE="ubuntu"
CONTAINER="otter-test-ubuntu-extra"
REPORT_FILE="test-report-ubuntu-extra-${BACKEND}-${MODE}.txt"
FAILURES=0

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------

ROOT_FLAG=""
if [ "${MODE}" = "rootful" ]; then
    ROOT_FLAG="--root"
fi

OTTER="otter"
CM_FLAGS="--container-manager ${BACKEND} ${ROOT_FLAG}"

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

printf "=== Integration Test: ubuntu-extra | backend=%s | mode=%s ===\n" \
    "${BACKEND}" "${MODE}" | tee "${REPORT_FILE}"
printf "Date: %s\n\n" "$(date -u)" | tee -a "${REPORT_FILE}"

# ----------------------------------------------------------------------------
# Setup
# ----------------------------------------------------------------------------

printf "\n--- Setup ---\n" | tee -a "${REPORT_FILE}"

run "registry pull" \
    "${OTTER} registry pull ${CM_FLAGS} ${IMAGE}"

run "create" \
    "${OTTER} create ${CM_FLAGS} ${CONTAINER} --image ${IMAGE}"

run "start" \
    "${OTTER} start ${CM_FLAGS} ${CONTAINER}"

# ----------------------------------------------------------------------------
# Lock / Unlock
# ----------------------------------------------------------------------------

printf "\n--- Lock / Unlock ---\n" | tee -a "${REPORT_FILE}"

run "lock basic" \
    "${OTTER} lock ${CM_FLAGS} ${CONTAINER}"

run "unlock basic" \
    "${OTTER} unlock ${CM_FLAGS} ${CONTAINER}"

run "lock all" \
    "${OTTER} lock ${CM_FLAGS} --all"

run "unlock all" \
    "${OTTER} unlock ${CM_FLAGS} --all"

# ----------------------------------------------------------------------------
# Pause / Restart
# ----------------------------------------------------------------------------

printf "\n--- Pause / Restart ---\n" | tee -a "${REPORT_FILE}"

run "pause basic" \
    "${OTTER} pause ${CM_FLAGS} ${CONTAINER}"

run "restart basic" \
    "${OTTER} restart ${CM_FLAGS} ${CONTAINER}"

run "pause all" \
    "${OTTER} pause ${CM_FLAGS} --all"

run "restart all" \
    "${OTTER} restart ${CM_FLAGS} --all"

# ----------------------------------------------------------------------------
# Teardown
# ----------------------------------------------------------------------------

printf "\n--- Teardown ---\n" | tee -a "${REPORT_FILE}"

run "remove" \
    "${OTTER} remove ${CM_FLAGS} ${CONTAINER} --force"

# ----------------------------------------------------------------------------
# Report
# ----------------------------------------------------------------------------

printf "\n=== Results: ubuntu-extra | backend=%s | mode=%s ===\n" \
    "${BACKEND}" "${MODE}" | tee -a "${REPORT_FILE}"

if [ "${FAILURES}" -eq 0 ]; then
    printf "All tests passed.\n" | tee -a "${REPORT_FILE}"
    exit 0
else
    printf "%d test(s) failed. See above for details.\n" "${FAILURES}" | tee -a "${REPORT_FILE}"
    exit 1
fi