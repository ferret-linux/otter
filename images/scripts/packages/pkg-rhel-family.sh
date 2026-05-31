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
	cracklib-dicts
	curl
	diffutils
	dnf-plugins-core
	findutils
	glibc-all-langpacks
	glibc-common
	glibc-locale-source
	gnupg2
	gnupg2-smime
	hostname
	iproute
	iputils
	keyutils
	krb5-libs
	less
	lsof
	man-db
	man-pages
	mesa-dri-drivers
	mesa-vulkan-drivers
	mtr
	ncurses
	nss-mdns
	openssh-clients
	pam
	passwd
	pigz
	pinentry
	procps-ng
	python3
	rsync
	shadow-utils
	sudo
	tcpdump
	time
	systemd
	traceroute
	tree
	tzdata
	unzip
	util-linux
	util-linux-script
	vte-profile
	vulkan
	wget
	which
	whois
	words
	xorg-x11-xauth
	xz
	zip
"

# Validate packages exist before installing
# shellcheck disable=SC2086,SC2046
dnf install -y $(dnf list -q ${deps} 2>/dev/null | \
	grep -v "Packages" | \
	grep "$(uname -m)" | \
	cut -d' ' -f1)