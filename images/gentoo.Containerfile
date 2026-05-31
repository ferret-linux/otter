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

# Sync portage tree
RUN emerge-webrsync

# Install default packages
RUN emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going \
    app-arch/pigz \
    app-arch/unzip \
    app-arch/zip \
    app-crypt/gnupg \
    app-crypt/pinentry \
    app-misc/tree \
    app-shells/bash \
    app-shells/bash-completion \
    dev-lang/python \
    dev-util/strace \
    dev-vcs/git \
    net-analyzer/mtr \
    net-analyzer/netcat \
    net-analyzer/socat \
    net-analyzer/tcpdump \
    net-analyzer/traceroute \
    net-misc/curl \
    net-misc/openssh \
    net-misc/rsync \
    net-misc/wget \
    sys-apps/diffutils \
    sys-apps/file \
    sys-apps/findutils \
    sys-apps/jq \
    sys-apps/keyutils \
    sys-apps/less \
    sys-apps/shadow \
    sys-apps/util-linux \
    sys-devel/bc \
    sys-libs/ncurses \
    sys-process/lsof \
    sys-process/procps \
    app-admin/sudo \
    x11-apps/xauth \
    x11-libs/vte

# Pre-generate locales
RUN sed -i 's|#.*en_US.UTF-8|en_US.UTF-8|g' /etc/locale.gen \
    && locale-gen \
    && printf 'LANG=en_US.UTF-8\nLC_CTYPE=en_US.UTF-8\n' > /etc/env.d/02locale \
    && env-update

# Pre-configure tzdata
RUN emerge --ask=n --noreplace --quiet-build --getbinpkg sys-libs/timezone-data \
    && ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/distfiles/* \
    /var/cache/binpkgs/* \
    /var/log/* \
    /var/tmp/*