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
	curl
	diffutils
	findutils
	glibc
	gnupg
	iputils
	inetutils
	keyutils
	less
	lsof
	man-db
	man-pages
	mlocate
	mesa
	mtr
	ncurses
	nss-mdns
	openssh
	pigz
	pinentry
	procps-ng
	python
	rsync
	shadow
	sudo
	tcpdump
	time
	traceroute
	tree
	tzdata
	unzip
	util-linux
	util-linux-libs
	vte-common
	vulkan-intel
	vulkan-radeon
	wget
	words
	xorg-xauth
	zip
"

# Validate packages exist before installing
# shellcheck disable=SC2086,SC2046
pacman -S --needed --noconfirm $(pacman -Ssq | grep -E "^($(echo ${deps} | tr ' ' '|'))$")