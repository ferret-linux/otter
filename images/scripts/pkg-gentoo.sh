#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

deps="
	app-crypt/gnupg
	app-crypt/pinentry
	app-shells/bash-completion
	app-text/tree
	dev-lang/python
	net-misc/curl
	net-misc/wget
	app-arch/pigz
	sys-apps/diffutils
	sys-apps/findutils
	sys-apps/less
	sys-apps/shadow
	sys-apps/openrc
	sys-apps/util-linux
	sys-devel/bc
	sys-libs/ncurses
	sys-process/lsof
	sys-process/procps
	app-admin/sudo
	x11-apps/xauth
"

# Validate packages exist before installing
install_pkg=""
for dep in ${deps}; do
	if [ "$(emerge --ask=n --search "${dep}" | grep "Applications found" | grep -Eo "[0-9]")" -gt 0 ]; then
		install_pkg="${install_pkg} ${dep}"
	fi
done

# shellcheck disable=SC2086
emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going ${install_pkg}