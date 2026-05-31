#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

deps="
	bash-completion
	bc
	bzip2
	curl
	diffutils
	findutils
	gnupg2
	inetutils
	iproute2
	less
	lsof
	man-db
	mesa-dri
	mesa-vulkan-intel
	mesa-vulkan-radeon
	mit-krb5
	mit-krb5-client
	mit-krb5-libs
	mtr
	ncurses
	nss
	openssh
	pigz
	pinentry
	pinentry-tty
	procps-ng
	python3
	rsync
	shadow
	sudo
	time
	traceroute
	tree
	tzdata
	unzip
	util-linux
	vulkan-loader
	vte3
	wget
	which
	xauth
	xz
	zip
"

# Validate packages exist before installing
# shellcheck disable=SC2086,SC2046
xbps-install -Sy $(xbps-query -Rs '*' | awk '{print $2}' | sed 's/-[^-]*$//' | grep -E "^($(echo ${deps} | tr ' ' '|'))$")