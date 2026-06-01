#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

set -e

# Add unique otter image marker
touch /usr/lib/otter/container.arch-family

# Configure pacman to install packages
# Configure pacman
sed -i "s|NoExtract.*||g" /etc/pacman.conf \
&& sed -i "s|NoProgressBar.*||g" /etc/pacman.conf \

# Install keyring first to ensure stuff works
pacman -Syy --noconfirm archlinux-keyring
# Update & upgrade base image packages
pacman -Syu --needed --noconfirm