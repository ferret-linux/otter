#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add generic otter image marker
touch /usr/lib/otter/container.official
# Add generic dirs to images
mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter \
    /usr/local/bin