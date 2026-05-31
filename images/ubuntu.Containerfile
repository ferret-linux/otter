# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ENV DEBIAN_FRONTEND=noninteractive

# Strip image cruft
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Install default packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-utils \
    bash \
    bash-completion \
    bc \
    bzip2 \
    ca-certificates \
    curl \
    dialog \
    diffutils \
    file \
    findutils \
    git \
    gnupg \
    gnupg2 \
    gpgsm \
    hostname \
    iproute2 \
    iputils-ping \
    jq \
    keyutils \
    language-pack-en \
    less \
    libcap2-bin \
    libegl-mesa0 \
    libegl1 \
    libgl1 \
    libglx-mesa0 \
    libkrb5-3 \
    libnss-mdns \
    libnss-myhostname \
    libvte-2.91-common \
    libvte-common \
    libvulkan1 \
    locales \
    lsof \
    man-db \
    manpages \
    mesa-vulkan-drivers \
    mtr \
    ncurses-base \
    netcat-openbsd \
    openssh-client \
    passwd \
    pigz \
    pinentry-curses \
    procps \
    python3 \
    rsync \
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
    wget \
    xauth \
    xz-utils \
    zip \
    && apt-get clean \
    && rm -rf /var/cache/apt/* \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/log/* \
    && rm -rf /var/tmp/*

# Pre-generate locales
RUN sed -i 's/^# *en_US.UTF-8/en_US.UTF-8/' /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata