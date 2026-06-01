# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create otter dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter \
    /usr/local/bin

# Add otter image identifiers
RUN touch /usr/lib/otter/container.official && \
    touch /usr/lib/otter/container.opensuse

# Configure zypper to install recommended packages and docs
RUN mkdir -p /etc/zypp/zypp.conf.d \
    && echo -e "[main]\nsolver.onlyRequires = false\nrpm.install.excludedocs = no" \
    > /etc/zypp/zypp.conf.d/99-otter.conf

# Lock parallel-printer-support as it can't be installed in rootless containers
RUN zypper al parallel-printer-support

# Add Packman repository for multimedia codecs
RUN . /etc/os-release && \
    if [ "${ID}" = "opensuse-tumbleweed" ]; then \
        zypper addrepo -cfp 90 'https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Tumbleweed/' packman; \
    elif [ "${ID}" = "opensuse-leap" ]; then \
        zypper addrepo -cfp 90 'https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Leap_$releasever/' packman; \
    fi && \
    zypper --gpg-auto-import-keys refresh && \
    zypper dist-upgrade --from packman --allow-vendor-change

# Upgrade all packages
RUN zypper dup -y --from packman --allow-vendor-change || zypper dup -y

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN zypper -n install -y $(sh /tmp/pkg-validator.sh --pkgmgr zypper -- \
    Mesa-dri \
    bash-completion \
    bc \
    bzip2 \
    curl \
    diffutils \
    findutils \
    glibc-locale \
    glibc-locale-base \
    gnupg \
    hostname \
    iputils \
    keyutils \
    less \
    libvulkan1 \
    libvulkan_intel \
    libvulkan_radeon \
    lsof \
    man \
    man-pages \
    mtr \
    ncurses \
    nss-mdns \
    openssh-clients \
    pam \
    pam-extra \
    pigz \
    systemd \
    pinentry \
    procps \
    glibc \
    glibc-i18n \
    glibc-i18ndata \
    glibc-locale-source \
    python3 \
    rsync \
    shadow \
    sudo \
    system-group-wheel \
    time \
    timezone \
    tree \
    unzip \
    util-linux \
    util-linux-systemd \
    wget \
    words \
    xauth \
    zip \
    pipewire \
    pipewire-pulseaudio \
    wireplumber \
    ffmpeg \
    gstreamer \
    gstreamer-plugins-base \
    gstreamer-plugins-good \
    gstreamer-plugins-bad \
    gstreamer-plugins-ugly \
    gstreamer-plugins-libav \
    libvpx2 \
    libx264-164 \
    libx265-215 \
    libopus0 \
    libFLAC12 \
    Mesa \
    Mesa-dri-devel \
    intel-media-driver \
    libva-intel-driver \
    mesa-libva \
    libva2) || [ ${?} = 106 ]

# Locale setup — glibc-locale (installed via pkg script) pre-builds locales on Leap;
# localedef requires glibc-i18ndata charmap files which are not available on Leap 15.x
RUN echo "LANG=en_US.UTF-8" > /etc/locale.conf

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN zypper clean --all \
    && rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*