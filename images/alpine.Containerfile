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
    bash \
    bc \
    fish \
    bash \
    zsh \
    bzip2 \
    coreutils \
    curl \
    ffmpeg \
    xdg-utils \
    xdg-user-dirs \
    gst-plugins-base \
    gst-plugins-bad \
    gst-plugins-ugly \
    gst-plugins-good \
    xdg-desktop-portal \
    pipewire \
    pipewire-jack \
    pipewire-pulse \
    vpl-gpu-rt \
    diffutils \
    findmnt \
    findutils \
    gnupg \
    gpg \
    iproute2 \
    iputils \
    keyutils \
    less \
    libcap \
    lsof \
    mount \
    ncurses \
    ncurses-terminfo \
    net-tools \
    openssh-client-default \
    pigz \
    pinentry \
    python3 \
    rsync \
    shadow \
    sudo \
    tcpdump \
    tree \
    tzdata \
    umount \
    openrc \
    unzip \
    util-linux \
    util-linux-login \
    util-linux-misc \
    vulkan-loader \
    wget \
    xauth \
    xz \
    zip)

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*