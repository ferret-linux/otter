#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./install-host-spawn.sh
# Downloads the pinned 1player/host-spawn release (static binary,
# amd64/arm64 only) and installs it to /usr/bin/host-spawn.
# Fails the build if the arch is unsupported or the download does not
# succeed.

set -e

HOST_SPAWN_VERSION="v1.6.2"

case "$(uname -m)" in
    x86_64) arch="x86_64" ;;
    aarch64) arch="aarch64" ;;
    *) echo "ERROR: unsupported architecture for host-spawn: $(uname -m)" >&2; exit 1 ;;
esac

url="https://github.com/1player/host-spawn/releases/download/${HOST_SPAWN_VERSION}/host-spawn-${arch}"

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

curl -fsSL -o "${tmp}/host-spawn" "${url}"

install -m 0755 "${tmp}/host-spawn" /usr/bin/host-spawn
