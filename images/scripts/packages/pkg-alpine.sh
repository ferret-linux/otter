#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

deps="
	bash
	bc
	bzip2
	coreutils
	curl
	diffutils
	findmnt
	findutils
	gnupg
	gpg
	iproute2
	iputils
	keyutils
	less
	libcap
	lsof
	mount
	ncurses
	ncurses-terminfo
	net-tools
	openssh-client-default
	pigz
	pinentry
	python3
	rsync
	shadow
	sudo
	tcpdump
	tree
	tzdata
	umount
	unzip
	util-linux
	util-linux-login
	util-linux-misc
	vulkan-loader
	wget
	xauth
	xz
	zip
"

# Validate packages exist before installing
# shellcheck disable=SC2086
found_deps="$(apk search -qe ${deps} | tr '\n' ' ')"
install_pkg=""
for dep in ${deps}; do
	case " ${found_deps} " in
		*" ${dep} "*)
			install_pkg="${install_pkg} ${dep}"
			;;
		*) ;;
	esac
done

# shellcheck disable=SC2086
apk add --force-overwrite ${install_pkg}