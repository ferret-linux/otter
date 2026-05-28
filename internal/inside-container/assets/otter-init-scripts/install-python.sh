#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-project/otter
# Copyright (C) 2026 otter contributors
#
# otter is free software; you can redistribute it and/or modify it
# under the terms of the GNU General Public License version 3
# as published by the Free Software Foundation.

if command -v apk > /dev/null 2>&1; then
	apk add python3
elif command -v apt-get > /dev/null 2>&1; then
	apt-get install -y python3
elif command -v dnf > /dev/null 2>&1; then
	dnf install -y python3
elif command -v microdnf > /dev/null 2>&1; then
	microdnf install -y python3
elif command -v yum > /dev/null 2>&1; then
	yum install -y python3
elif command -v zypper > /dev/null 2>&1; then
	zypper install -y python3
elif command -v pacman > /dev/null 2>&1; then
	pacman -S --needed --noconfirm python
elif command -v emerge > /dev/null 2>&1; then
	emerge --ask=n --noreplace dev-lang/python
elif command -v xbps-install > /dev/null 2>&1; then
	xbps-install -Sy python3
elif command -v slackpkg > /dev/null 2>&1; then
	yes | slackpkg install -default_answer=yes -batch=yes python3
else
	printf "Error: could not find a supported package manager to install Python.\n"
	exit 127
fi