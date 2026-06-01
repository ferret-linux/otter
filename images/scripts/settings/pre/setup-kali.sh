#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.kali

# Fix dpkg to install packages
rm -f /etc/dpkg/dpkg.cfg.d/excludes
# Update & upgrade base image packages
apt update && apt upgrade --yes && apt autoremove --purge --yes