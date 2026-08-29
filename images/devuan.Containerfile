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
RUN touch /usr/lib/otter/container.devuan

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Upgrade all packages, install the complete package set, perform
# Apt cleanup, and remove Apt/filesystem caches before this layer
# is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
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
        unrar \
        rsync \
        unzip \
        xauth \
        dialog \
        gnupg2 \
        ffmpeg \
        libgl1 \
        man-db \
        passwd \
        procps \
        tzdata \
        libvpx9 \
        libegl1 \
        locales \
        python3 \
        tcpdump \
        hostname \
        iproute2 \
        keyutils \
        sysvinit \
        pipewire \
        libopus0 \
        manpages \
        xz-utils \
        xwayland \
        apt-utils \
        diffutils \
        findutils \
        libflac12 \
        libvdpau1 \
        libkrb5-3 \
        xdg-utils \
        libvulkan1 \
        traceroute \
        util-linux \
        libcap2-bin \
        wireplumber \
        libx264-164 \
        libx265-199 \
        libmp3lame0 \
        libnss-mdns \
        iputils-ping \
        libegl-mesa0 \
        libglx-mesa0 \
        ncurses-base \
        xdg-user-dirs \
        libvte-common \
        pipewire-alsa \
        pipewire-jack \
        pipewire-pulse \
        openssh-client \
        bash-completion \
        libgl1-mesa-dri \
        libgl1-mesa-glx \
        pinentry-curses \
        mesa-va-drivers \
        libavcodec-extra \
        libpipewire-0.3-0 \
        libnss-myhostname \
        xdg-desktop-portal \
        gstreamer1.0-tools \
        gstreamer1.0-libav \
        mesa-vulkan-drivers \
        gstreamer1.0-pipewire \
        gstreamer1.0-plugins-bad \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        intel-media-va-driver-non-free) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/tmp/*

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

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