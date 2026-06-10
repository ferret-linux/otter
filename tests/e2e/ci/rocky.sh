#!/usr/bin/env bash
# shellcheck shell=bash
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./rocky.sh <runtime> <mode>
#   runtime: podman | docker | nerdctl
#   mode:    rootless | rootful

set -euo pipefail

RUNTIME="${1:-podman}"
MODE="${2:-rootless}"
CONTAINER_NAME="otter-test-rocky"
IMAGE="rocky"
OTTER=(otter --container-manager="${RUNTIME}")
ROOT_FLAG=()

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

FAILURES=()

if [[ "${MODE}" == "rootful" ]]; then
    ROOT_FLAG=(--root)
fi

if [[ "${RUNTIME}" == "nerdctl" && "${MODE}" == "rootless" ]]; then
    echo "nerdctl rootless is not supported, skipping."
    exit 0
fi

pass() { printf '%b✔ %s%b\n' "${GREEN}" "${1}" "${RESET}"; }
fail() { printf '%b✘ %s%b\n' "${RED}" "${1}" "${RESET}"; FAILURES+=("${1}"); }

# shellcheck disable=SC2329
cleanup() {
    "${OTTER[@]}" remove "${ROOT_FLAG[@]}" --force "${CONTAINER_NAME}" 2>/dev/null || true
}
trap cleanup EXIT

run_step() {
    local name="${1}"
    shift
    if "$@" > /dev/null 2>&1; then
        pass "${name}"
    else
        fail "${name}"
    fi
}

printf "==> rocky | runtime=%s | mode=%s\n\n" "${RUNTIME}" "${MODE}"

run_step "registry pull" "${OTTER[@]}" reg pull "${ROOT_FLAG[@]}" "${IMAGE}"
run_step "create"        "${OTTER[@]}" create "${ROOT_FLAG[@]}" --image "${IMAGE}" "${CONTAINER_NAME}"
run_step "start"         "${OTTER[@]}" start "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "list"          "${OTTER[@]}" list "${ROOT_FLAG[@]}"
run_step "info"          "${OTTER[@]}" inspect "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "logs"          "${OTTER[@]}" journal "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "upgrade"       "${OTTER[@]}" upgrade "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "restart"       "${OTTER[@]}" restart "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "pause"         "${OTTER[@]}" pause "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "lock"          "${OTTER[@]}" lock "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "unlock"        "${OTTER[@]}" unlock "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "stop"          "${OTTER[@]}" stop "${ROOT_FLAG[@]}" "${CONTAINER_NAME}"
run_step "remove"        "${OTTER[@]}" remove "${ROOT_FLAG[@]}" --force "${CONTAINER_NAME}"

echo
if [[ ${#FAILURES[@]} -eq 0 ]]; then
    printf '%ball steps passed%b\n' "${GREEN}" "${RESET}"
    exit 0
else
    printf '%bfailed steps: %s%b\n' "${RED}" "${FAILURES[*]}" "${RESET}"
    exit 1
fi