# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Strip image cruft
RUN [ -e /etc/dnf/dnf.conf ] && sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf || true \
    && [ -e /etc/yum.conf ] && sed -i '/tsflags=nodocs/d' /etc/yum.conf || true

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Install default packages
RUN dnf install -y \
    bash \
    bash-completion \
    bc \
    bzip2 \
    ca-certificates \
    cracklib-dicts \
    curl \
    diffutils \
    file \
    findutils \
    git \
    glibc-all-langpacks \
    glibc-common \
    glibc-locale-source \
    gnupg2 \
    gnupg2-smime \
    hostname \
    iproute \
    iputils \
    jq \
    keyutils \
    krb5-libs \
    less \
    lsof \
    man-db \
    man-pages \
    mesa-dri-drivers \
    mesa-vulkan-drivers \
    mtr \
    ncurses \
    nss-mdns \
    openssh-clients \
    pam \
    passwd \
    pigz \
    pinentry \
    procps-ng \
    python3 \
    rsync \
    shadow-utils \
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
    util-linux-script \
    vte-profile \
    wget \
    which \
    whois \
    words \
    xorg-x11-xauth \
    xz \
    zip \
    && dnf clean all

# Pre-generate locales
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/dnf/* \
    /var/cache/yum/* \
    /var/log/* \
    /var/tmp/*