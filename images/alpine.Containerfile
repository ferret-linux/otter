# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Install default packages
RUN apk add --no-cache \
    bash \
    bash-completion \
    bc \
    bzip2 \
    ca-certificates \
    coreutils \
    curl \
    diffutils \
    file \
    findmnt \
    findutils \
    git \
    gnupg \
    gpg \
    iproute2 \
    iputils \
    jq \
    keyutils \
    less \
    libcap \
    lsof \
    man-db \
    man-pages \
    mount \
    ncurses \
    ncurses-terminfo \
    net-tools \
    netcat-openbsd \
    openssh-client-default \
    pigz \
    pinentry \
    procps \
    python3 \
    rsync \
    shadow \
    socat \
    strace \
    sudo \
    tcpdump \
    tmux \
    tree \
    tzdata \
    umount \
    unzip \
    util-linux \
    util-linux-login \
    util-linux-misc \
    vulkan-loader \
    wget \
    xauth \
    xz \
    zip

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*