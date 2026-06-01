#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.opensuse

# Configure zypper to install recommended packages and docs
mkdir -p /etc/zypp/zypp.conf.d
cat > /etc/zypp/zypp.conf.d/99-otter.conf << 'EOF'
[main]
solver.onlyRequires = false
rpm.install.excludedocs = no
EOF
# Lock parallel-printer-support as it can't be installed in rootless containers
zypper al parallel-printer-support
# Update & upgrade base image packages
zypper dup -y