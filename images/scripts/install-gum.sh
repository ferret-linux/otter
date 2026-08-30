#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./install-gum.sh
# Downloads the latest charmbracelet/gum release (static binary,
# amd64/arm64 only) and installs it to /usr/lib/otter/helpers/gum.
# Verifies the download against the release's published checksums.txt.
# Fails the build if the arch is unsupported or the download/checksum
# does not succeed (no pinned-version fallback).

set -e

case "$(uname -m)" in
    x86_64) arch="x86_64" ;;
    aarch64) arch="arm64" ;;
    *) echo "ERROR: unsupported architecture for gum: $(uname -m)" >&2; exit 1 ;;
esac

tag="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/charmbracelet/gum/releases/latest | sed 's#.*/tag/##')"
[ -n "${tag}" ]
version="${tag#v}"

tarball="gum_${version}_Linux_${arch}.tar.gz"
base_url="https://github.com/charmbracelet/gum/releases/download/${tag}"

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

curl -fsSL -o "${tmp}/${tarball}" "${base_url}/${tarball}"
curl -fsSL -o "${tmp}/checksums.txt" "${base_url}/checksums.txt"

grep "  ${tarball}\$" "${tmp}/checksums.txt" > "${tmp}/checksums.filtered"
[ -s "${tmp}/checksums.filtered" ]

( cd "${tmp}" && sha256sum -c checksums.filtered )

tar -xzf "${tmp}/${tarball}" -C "${tmp}"

bin="$(find "${tmp}" -type f -name gum | head -n1)"
[ -n "${bin}" ]

install -m 0755 "${bin}" /usr/lib/otter/helpers/gum
