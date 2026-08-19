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
RUN touch /usr/lib/otter/container.ubuntu

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Accept microsoft fonts eula
RUN echo "ttf-mscorefonts-installer msttcorefonts/accepted-mscorefonts-eula select true" | debconf-set-selections

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Upgrade the system, install the requested packages, and clean
# APT metadata/cache in the same layer.
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        bc \
        zsh \
        mtr \
        zip \
        curl \
        fish \
        bash \
        less \
        lsof \
        pigz \
        sudo \
        time \
        tree \
        wget \
        bzip2 \
        gnupg \
        gpgsm \
        rsync \
        unzip \
        xauth \
        dialog \
        gnupg2 \
        libgl1 \
        man-db \
        passwd \
        procps \
        tzdata \
        systemd \
        libegl1 \
        locales \
        python3 \
        tcpdump \
        hostname \
        iproute2 \
        keyutils \
        manpages \
        xz-utils \
        apt-utils \
        diffutils \
        findutils \
        libkrb5-3 \
        libvulkan1 \
        traceroute \
        util-linux \
        libcap2-bin \
        libnss-mdns \
        iputils-ping \
        libegl-mesa0 \
        libegl1-mesa \
        libglx-mesa0 \
        ncurses-base \
        libvte-common \
        openssh-client \
        bash-completion \
        libgl1-mesa-glx \
        pinentry-curses \
        language-pack-en \
        libnss-myhostname \
        mesa-vulkan-drivers \
        ubuntu-restricted-extras \
        ubuntu-restricted-addons) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean && \
    rm -rf \
    /var/lib/apt/lists/* \
    /var/log/* \
    /var/tmp/*

# Locale setup
RUN sed -i "s|# en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh
