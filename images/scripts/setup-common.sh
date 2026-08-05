#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./setup-common.sh
# it's used to setup common settings/defaults that are generic across distros

# Make common otter specific dir's
mkdir -p \
    /usr/libexec/ \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter \
    /usr/local/bin \
    /usr/lib/otter/scripts \
    /usr/lib/otter/helpers 

# Add generic image identifier
touch /usr/lib/otter/container.official