#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.gentoo

# Make portage use pre-compiled bins
echo 'FEATURES="getbinpkg"' >> /etc/portage/make.conf
# sync portage & fetch package keys
emerge-webrsync && getuto
# Update & upgrade base image packages
emerge --ask=n --autounmask-continue --quiet-build --getbinpkg -uDN @world