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

# Upgrade all packages
RUN apt-get update && apt-get upgrade -y

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
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
    apt-utils \
    diffutils \
    findutils \
    libflac12 \
    libvdpau1 \
    libkrb5-3 \
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
    pipewire-pulse \
    openssh-client \
    bash-completion \
    libgl1-mesa-dri \
    libgl1-mesa-glx \
    pinentry-curses \
    libavcodec-extra \
    libpipewire-0.3-0 \
    libnss-myhostname \
    gstreamer1.0-tools \
    gstreamer1.0-libav \
    mesa-vulkan-drivers \
    gstreamer1.0-plugins-bad \
    gstreamer1.0-plugins-base \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-ugly \
    intel-media-va-driver-non-free)

# Locale setup
RUN sed -i "s|# en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN apt-get clean \
    && rm -rf \
    /var/lib/apt/lists/* \
    /var/log/* \
    /var/tmp/*