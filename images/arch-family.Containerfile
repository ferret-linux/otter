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

# Install packages
RUN pacman -S --needed --noconfirm \
    bash bash-completion bc curl diffutils findutils glibc gnupg \
    inetutils iputils keyutils less lsof man-db man-pages mesa mlocate \
    mtr ncurses nss-mdns openssh pigz pinentry procps-ng python rsync \
    shadow sudo tcpdump time traceroute tree tzdata unzip util-linux \
    util-linux-libs vte-common vulkan-intel vulkan-radeon wget words \
    xorg-xauth zip systemd base-devel git tar curl wget xdg-utils

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/pacman/pkg/* \
    /var/log/* \
    /var/tmp/*