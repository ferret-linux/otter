#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./install-starship.sh
# Downloads the latest starship/starship release (static musl binary,
# amd64/arm64 only) and installs it to /usr/local/bin/starship.
# Verifies the download against the release's published per-asset
# .sha256 checksum. Fails the build if the arch is unsupported or the
# download/checksum does not succeed (no pinned-version fallback).

set -e

case "$(uname -m)" in
	x86_64) arch="x86_64" ;;
	aarch64) arch="aarch64" ;;
	*) echo "ERROR: unsupported architecture for starship: $(uname -m)" >&2; exit 1 ;;
esac

tag="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/starship/starship/releases/latest | sed 's#.*/tag/##')"
[ -n "${tag}" ]

tarball="starship-${arch}-unknown-linux-musl.tar.gz"
base_url="https://github.com/starship/starship/releases/download/${tag}"

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

curl -fsSL -o "${tmp}/${tarball}" "${base_url}/${tarball}"
curl -fsSL -o "${tmp}/${tarball}.sha256" "${base_url}/${tarball}.sha256"

echo "$(cat "${tmp}/${tarball}.sha256")  ${tarball}" > "${tmp}/checksum.filtered"

( cd "${tmp}" && sha256sum -c checksum.filtered )

tar -xzf "${tmp}/${tarball}" -C "${tmp}"

bin="$(find "${tmp}" -type f -name starship | head -n1)"
[ -n "${bin}" ]

install -m 0755 "${bin}" /usr/local/bin/starship
echo "starship: installed $(/usr/local/bin/starship --version | head -n1)"
