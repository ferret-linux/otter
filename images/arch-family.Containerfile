# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Strip image cruft — enable locale, docs, color
RUN sed -i 's|NoExtract.*||g' /etc/pacman.conf \
    && sed -i 's|NoProgressBar.*||g' /etc/pacman.conf \
    && sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Install default packages
RUN pacman -Syu --noconfirm && pacman -S --needed --noconfirm \
    bash \
    bash-completion \
    bc \
    ca-certificates \
    curl \
    diffutils \
    file \
    findutils \
    git \
    glibc \
    gnupg \
    inetutils \
    iputils \
    jq \
    keyutils \
    less \
    lsof \
    man-db \
    man-pages \
    mesa \
    mlocate \
    mtr \
    ncurses \
    nss-mdns \
    openssh \
    openbsd-netcat \
    pigz \
    pinentry \
    procps-ng \
    python \
    rsync \
    shadow \
    socat \
    strace \
    sudo \
    tcpdump \
    time \
    tmux \
    traceroute \
    tree \
    tzdata \
    unzip \
    util-linux \
    util-linux-libs \
    vte-common \
    vulkan-intel \
    vulkan-radeon \
    wget \
    words \
    xorg-xauth \
    zip \
    && pacman -Scc --noconfirm

# Pre-generate locales
RUN sed -i 's|#.*en_US.UTF-8|en_US.UTF-8|g' /etc/locale.gen \
    && locale-gen

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/pacman/* \
    /var/log/* \
    /var/tmp/*