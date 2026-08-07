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
RUN touch /usr/lib/otter/container.opensuse

# Configure zypper to install recommended packages and docs
RUN mkdir -p /etc/zypp/zypp.conf.d \
    && echo -e "[main]\nsolver.onlyRequires = false\nrpm.install.excludedocs = no" \
    > /etc/zypp/zypp.conf.d/99-otter.conf

# Lock parallel-printer-support as it can't be installed in rootless containers
RUN zypper al parallel-printer-support

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN zypper -n install --auto-agree-with-licenses -y \
    $(sh /tmp/pkg-validator.sh --pkgmgr zypper -- \
    bc \
    zsh \
    man \
    mtr \
    pam \
    zip \
    fish \
    bash \
    curl \
    less \
    lsof \
    pigz \
    sudo \
    time \
    tree \
    wget \
    Mesa \
    bzip2 \
    gnupg \
    glibc \
    rsync \
    unzip \
    words \
    xauth \
    procps \
    shadow \
    ffmpeg \
    libva2 \
    iputils \
    ncurses \
    systemd \
    python3 \
    libvpx2 \
    Mesa-dri \
    hostname \
    keyutils \
    nss-mdns \
    pinentry \
    timezone \
    pipewire \
    libopus0 \
    diffutils \
    findutils \
    man-pages \
    pam-extra \
    gstreamer \
    libFLAC12 \
    libvulkan1 \
    glibc-i18n \
    util-linux \
    mesa-libva \
    wireplumber \
    libx264-164 \
    libx265-215 \
    glibc-locale \
    glibc-i18ndata \
    Mesa-dri-devel \
    bash-completion \
    libvulkan_intel \
    openssh-clients \
    libvulkan_radeon \
    glibc-locale-base \
    system-group-wheel \
    util-linux-systemd \
    intel-media-driver \
    libva-intel-driver \
    glibc-locale-source \
    pipewire-pulseaudio \
    gstreamer-plugins-bad \
    gstreamer-plugins-base \
    gstreamer-plugins-good \
    gstreamer-plugins-ugly \
    gstreamer-plugins-libav) || { rc=${?}; [ "${rc}" -ge 100 ] && [ "${rc}" -lt 200 ] && [ "${rc}" -ne 105 ]; }

# Locale setup — glibc-locale (installed via pkg script) pre-builds locales on Leap;
# localedef requires glibc-i18ndata charmap files which are not available on Leap 15.x
RUN echo "LANG=en_US.UTF-8" > /etc/locale.conf

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Zypper cleanup
RUN zypper refresh && \
    zypper update -y && \
    zypper rm $(zypper packages --orphaned -i | awk -F'|' '/^i/{print $3}') && \
    zypper clean

# Cleanup
RUN zypper clean --all \
    && rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*