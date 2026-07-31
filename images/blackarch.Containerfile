# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.blackarch

# Configure pacman
RUN sed -i "s|NoExtract.*||g" /etc/pacman.conf \
    && sed -i "s|NoProgressBar.*||g" /etc/pacman.conf \
    && sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

# Upgrade all packages
RUN pacman -Syyu --noconfirm

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
    bc \
    zsh \
    git \
    tar \
    mtr \
    zip \
    fish \
    bash \
    curl \
    less \
    lsof \
    mesa \
    pigz \
    sudo \
    time \
    tree \
    wget \
    glibc \
    gnupg \
    rsync \
    unzip \
    words \
    ffmpeg \
    man-db \
    python \
    shadow \
    tzdata \
    systemd \
    iputils \
    mlocate \
    ncurses \
    openssh \
    tcpdump \
    pipewire \
    fakeroot \
    keyutils \
    nss-mdns \
    pinentry \
    xdg-utils \
    diffutils \
    findutils \
    inetutils \
    man-pages \
    procps-ng \
    vpl-gpu-rt \
    base-devel \
    traceroute \
    util-linux \
    vte-common \
    xorg-xauth \
    vulkan-intel \
    pipewire-jack \
    xdg-user-dirs \
    vulkan-radeon \
    pipewire-pulse \
    bash-completion \
    gst-plugins-bad \
    util-linux-libs \
    gst-plugins-base \
    gst-plugins-good \
    gst-plugins-ugly \
    xdg-desktop-portal)

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/pacman/pkg/* \
    /var/log/* \
    /var/tmp/*