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
RUN touch /usr/lib/otter/container.chimera

# Upgrade all packages
RUN apk update && apk upgrade
RUN apk add base-bootstrap && apk upgrade -Ua

# Enable Chimera's user repo for additional packages
RUN apk add chimera-repo-user && apk update

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
    bc \
    xz \
    zsh \
    gpg \
    vte \
    zip \
    bash \
    fish \
    curl \
    gtar \
    less \
    lsof \
    pigz \
    sudo \
    tree \
    wget \
    which \
    bc-gh \
    bzip2 \
    gnupg \
    mount \
    rsync \
    dinit \
    unzip \
    wget2 \
    xauth \
    ffmpeg \
    libcap \
    shadow \
    tzdata \
    umount \
    findmnt \
    iputils \
    ncurses \
    openssh \
    python3 \
    tcpdump \
    pipewire \
    iproute2 \
    keyutils \
    mesa-dri \
    opendoas \
    pinentry \
    coreutils \
    xdg-utils \
    diffutils \
    findutils \
    net-tools \
    vpl-gpu-rt \
    util-linux \
    libcap-progs \
    ncurses-term \
    base-full-man \
    xdg-user-dirs \
    pipewire-jack \
    dinit-chimera \
    vulkan-loader \
    pipewire-pulse \
    base-bootstrap \
    bash-completion \
    gst-plugins-bad \
    util-linux-misc \
    gst-plugins-base \
    gst-plugins-ugly \
    gst-plugins-good \
    libarchive-progs \
    ncurses-terminfo \
    util-linux-login \
    util-linux-mount \
    xdg-desktop-portal)

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Apk cleanup
RUN apk update && \
    apk upgrade && \
    apk cache clean

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*