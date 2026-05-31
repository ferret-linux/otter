#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

deps="
	Mesa-dri
	bash-completion
	bc
	bzip2
	curl
	diffutils
	findutils
	glibc-locale
	glibc-locale-base
	gnupg
	hostname
	iputils
	keyutils
	less
	libvulkan1
	libvulkan_intel
	libvulkan_radeon
	lsof
	man
	man-pages
	mtr
	ncurses
	nss-mdns
	openssh-clients
	pam
	pam-extra
	pigz
	systemd
	pinentry
	procps
	glibc-i18n
	glibc-locale
	glibc-i18ndata
	python3
	rsync
	shadow
	sudo
	system-group-wheel
	systemd
	time
	timezone
	tree
	unzip
	util-linux
	util-linux-systemd
	wget
	words
	xauth
	zip
"

# Validate packages exist before installing — mark gpg errors (106) as non-fatal
# shellcheck disable=SC2086,SC2046
zypper -n install -y $(zypper -n -q se --match-exact ${deps} | grep -e 'package$' | cut -d'|' -f2) || [ ${?} = 106 ]