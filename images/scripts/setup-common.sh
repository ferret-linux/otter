#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./setup-common.sh
# it's used to setup common settings/defaults that are generic across distros

# Remove broken dir-files from images
rm -rf /opt /root

# Make common otter specific dir's
mkdir -p \
    /opt \
    /run \
    /tmp \
    /var \
    /root \
    /home \
    /usr/libexec/ \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter \
    /usr/local/bin \
    /usr/lib/otter/scripts \
    /usr/lib/otter/helpers 

# Fix Permissions for dirs
chmod 700 /root
chmod 755 /opt
chmod 755 /usr/local/bin
chmod 755 /usr/libexec
chmod 755 /etc/profile.d
chmod 750 /etc/sudoers.d
chmod 1777 /tmp /var/tmp

# Add generic image identifier
touch /usr/lib/otter/container.official