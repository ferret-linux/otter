#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

export DEBIAN_FRONTEND=noninteractive

deps="
	apt-utils
	bash-completion
	bc
	bzip2
	curl
	dialog
	diffutils
	findutils
	systemd
	gnupg
	gnupg2
	gpgsm
	hostname
	iproute2
	iputils-ping
	keyutils
	language-pack-en
	less
	libcap2-bin
	libkrb5-3
	libegl-mesa0
	libegl1
	libgl1
	libgl1-mesa-glx
	libglx-mesa0
	libnss-mdns
	libnss-myhostname
	libvulkan1
	locales
	lsof
	man-db
	manpages
	mesa-vulkan-drivers
	mtr
	ncurses-base
	openssh-client
	passwd
	pigz
	pinentry-curses
	procps
	python3
	rsync
	sudo
	ubuntu-restricted-extras
	ubuntu-restricted-addons
	systemd
	tcpdump
	time
	traceroute
	tree
	tzdata
	unzip
	util-linux
	wget
	xauth
	xz-utils
	zip
"

# Only install packages that exist in the repo — mirrors otter-init's setup_apt pattern
# shellcheck disable=SC2086,SC2046
apt-get install -y --no-install-suggests \
	$(apt-cache show ${deps} 2>/dev/null | grep "^Package:" | sort -u | cut -d' ' -f2-)