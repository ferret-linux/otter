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
    /etc/xbps.d \
    /usr/lib/otter

# Pre-create xbps ignore conf to prevent overwriting critical files
RUN printf 'noextract=/etc/passwd\nnoextract=/etc/hosts\nnoextract=/etc/host.conf\nnoextract=/etc/hostname\nnoextract=/etc/localtime\nnoextract=/etc/machine-id\nnoextract=/etc/resolv.conf\n' \
    > /etc/xbps.d/otter-ignore.conf

# Install default packages
RUN xbps-install -Syu xbps \
    && xbps-install -Sy \
    bash \
    bash-completion \
    bc \
    bzip2 \
    ca-certificates \
    curl \
    diffutils \
    file \
    findutils \
    git \
    gnupg2 \
    inetutils \
    iproute2 \
    jq \
    less \
    lsof \
    man-db \
    mesa-dri \
    mesa-vulkan-intel \
    mesa-vulkan-radeon \
    mit-krb5 \
    mit-krb5-client \
    mit-krb5-libs \
    mtr \
    ncurses \
    netcat-openbsd \
    nss \
    openssh \
    pigz \
    pinentry \
    pinentry-tty \
    procps-ng \
    python3 \
    rsync \
    shadow \
    socat \
    strace \
    sudo \
    time \
    tmux \
    traceroute \
    tree \
    tzdata \
    unzip \
    util-linux \
    vte3 \
    vulkan-loader \
    wget \
    which \
    xauth \
    xz \
    zip \
    && xbps-remove -Oo

# Pre-generate locales (glibc only, musl has no locale system)
RUN if [ -f /etc/default/libc-locales ]; then \
        sed -i 's|#.*en_US.UTF-8|en_US.UTF-8|g' /etc/default/libc-locales \
        && xbps-reconfigure --force glibc-locales; \
    fi

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/xbps/* \
    /var/log/* \
    /var/tmp/*