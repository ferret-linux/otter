#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.fedora

# Re-enable docs
sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || true
# Update & upgrade base image packages
dnf upgrade -y --refresh