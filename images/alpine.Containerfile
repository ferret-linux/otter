# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.alpine

# Upgrade all packages
RUN apk update && apk upgrade

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
    bc \
    xz \
    tar \
    zsh \
    gpg \
    zip \
    bash \
    fish \
    bash \
    curl \
    docs \
    less \
    lsof \
    pigz \
    sudo \
    tree \
    vte3 \
    wget \
    which \
    bzip2 \
    gnupg \
    mount \
    rsync \
    unzip \
    xauth \
    ffmpeg \
    libcap \
    shadow \
    mandoc \
    tzdata \
    umount \
    openrc \
    gcompat \
    findmnt \
    iputils \
    ncurses \
    python3 \
    tcpdump \
    pipewire \
    iproute2 \
    keyutils \
    pinentry \
    man-pages \
    coreutils \
    xdg-utils \
    diffutils \
    findutils \
    net-tools \
    vpl-gpu-rt \
    util-linux \
    musl-utils \
    alpine-base \
    xdg-user-dirs \
    pipewire-jack \
    vulkan-loader \
    pipewire-pulse \
    bash-completion \
    gst-plugins-bad \
    util-linux-misc \
    gst-plugins-base \
    gst-plugins-ugly \
    gst-plugins-good \
    ncurses-terminfo \
    util-linux-login \
    xdg-desktop-portal \
    openssh-client-default)

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*