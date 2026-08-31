#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./install-shell.sh <fish|nu> [<fish|nu> ...]
# Installs one or more of fish/nu as a static/musl binary (amd64/arm64
# only) if it isn't already available on PATH -- for distros whose
# package manager doesn't carry it. Mirrors install-gum.sh's shape
# (arch detection, tmp dir, checksum where upstream provides one).
# nu's build is always the musl variant regardless of host libc, since
# it's the more portable, distro-independent option.

set -e

case "$(uname -m)" in
	x86_64) arch="x86_64" ;;
	aarch64) arch="aarch64" ;;
	*) echo "ERROR: unsupported architecture for shell install: $(uname -m)" >&2; exit 1 ;;
esac

install_fish()
{
	if command -v fish > /dev/null; then
		echo "fish: already installed ($(fish --version))"
		return
	fi
	echo "fish: not installed, fetching latest release"

	tag="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/fish-shell/fish-shell/releases/latest | sed 's#.*/tag/##')"
	[ -n "${tag}" ]
	version="${tag#v}"

	tarball="fish-${version}-linux-${arch}.tar.xz"
	base_url="https://github.com/fish-shell/fish-shell/releases/download/${tag}"

	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp}"' EXIT

	curl -fsSL -o "${tmp}/${tarball}" "${base_url}/${tarball}"

	tar -xJf "${tmp}/${tarball}" -C "${tmp}"

	bin="$(find "${tmp}" -type f -name fish | head -n1)"
	[ -n "${bin}" ]

	install -m 0755 "${bin}" /usr/local/bin/fish
	echo "fish: installed $(/usr/local/bin/fish --version)"

	rm -rf "${tmp}"
	trap - EXIT
}

install_nu()
{
	if command -v nu > /dev/null; then
		echo "nu: already installed ($(nu --version))"
		return
	fi
	echo "nu: not installed, fetching latest release (musl)"

	tag="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/nushell/nushell/releases/latest | sed 's#.*/tag/##')"
	[ -n "${tag}" ]
	version="${tag#v}"

	tarball="nu-${version}-${arch}-unknown-linux-musl.tar.gz"
	base_url="https://github.com/nushell/nushell/releases/download/${tag}"

	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp}"' EXIT

	curl -fsSL -o "${tmp}/${tarball}" "${base_url}/${tarball}"
	curl -fsSL -o "${tmp}/SHA256SUMS" "${base_url}/SHA256SUMS"

	grep "  ${tarball}\$" "${tmp}/SHA256SUMS" > "${tmp}/checksums.filtered"
	[ -s "${tmp}/checksums.filtered" ]

	( cd "${tmp}" && sha256sum -c checksums.filtered )

	tar -xzf "${tmp}/${tarball}" -C "${tmp}"

	bin="$(find "${tmp}" -type f -name nu | head -n1)"
	[ -n "${bin}" ]

	install -m 0755 "${bin}" /usr/local/bin/nu
	echo "nu: installed $(/usr/local/bin/nu --version | head -n1)"

	rm -rf "${tmp}"
	trap - EXIT
}

if [ "$#" -eq 0 ]; then
	echo "ERROR: no shell specified. Usage: install-shell.sh <fish|nu> [...]" >&2
	exit 1
fi

for requested in "$@"; do
	case "${requested}" in
		fish) install_fish ;;
		nu) install_nu ;;
		*) echo "ERROR: install-shell.sh does not support '${requested}' (only fish, nu)" >&2; exit 1 ;;
	esac
done