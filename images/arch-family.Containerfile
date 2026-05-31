# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create otter dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Configure pacman
RUN sed -i "s|NoExtract.*||g" /etc/pacman.conf \
    && sed -i "s|NoProgressBar.*||g" /etc/pacman.conf \
    && sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

# Upgrade all packages
RUN pacman -Syyu --noconfirm

# Run package install script
COPY images/scripts/pkg_arch_family.sh /tmp/pkg_arch_family.sh
RUN sh /tmp/pkg_arch_family.sh

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/pacman/pkg/* \
    /var/log/* \
    /var/tmp/*