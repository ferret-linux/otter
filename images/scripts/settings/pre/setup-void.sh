#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.void

# Configure xbps to not overwrite host-managed files
mkdir -p /etc/xbps.d
cat > /etc/xbps.d/otter-ignore.conf << 'EOF'
noextract=/etc/passwd
noextract=/etc/hosts
noextract=/etc/host.conf
noextract=/etc/hostname
noextract=/etc/localtime
noextract=/etc/machine-id
noextract=/etc/resolv.conf
EOF

# Update xbps itself first
xbps-install -Syu xbps
# Update & upgrade base image packages
xbps-install -Syu