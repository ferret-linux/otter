# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Strip image cruft — enable recommended packages and docs
RUN if [ -d /etc/zypp/zypp.conf.d ]; then \
        printf '[main]\nsolver.onlyRequires = false\nrpm.install.excludedocs = no\n' \
        > /etc/zypp/zypp.conf.d/99-otter.conf; \
    else \
        sed -i 's/.*solver.onlyRequires.*/solver.onlyRequires = false/g' /etc/zypp/zypp.conf && \
        sed -i 's/.*rpm.install.excludedocs.*/rpm.install.excludedocs = no/g' /etc/zypp/zypp.conf; \
    fi

# Pre-create default dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Lock parallel-printer-support — can't install in rootless containers
RUN zypper al parallel-printer-support

# Install default packages
RUN zypper --non-interactive install \
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
    glibc-locale \
    glibc-locale-base \
    gnupg \
    hostname \
    iputils \
    jq \
    keyutils \
    less \
    libvulkan1 \
    libvulkan_intel \
    libvulkan_radeon \
    lsof \
    man \
    man-pages \
    Mesa-dri \
    mtr \
    ncurses \
    netcat-openbsd \
    nss-mdns \
    openssh-clients \
    pam \
    pam-extra \
    pigz \
    pinentry \
    procps \
    python3 \
    rsync \
    shadow \
    socat \
    strace \
    sudo \
    system-group-wheel \
    systemd \
    time \
    timezone \
    tmux \
    tree \
    unzip \
    util-linux \
    util-linux-systemd \
    wget \
    words \
    xauth \
    xz \
    zip \
    && zypper clean --all

# Pre-generate locales
RUN localedef -i en_US -f UTF-8 en_US.UTF-8 || true

# Pre-configure tzdata
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*