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
    touch /usr/lib/otter/container.rhel-family

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || \
    sed -i '/tsflags=nodocs/d' /etc/yum.conf 2>/dev/null || true


# Install epel repos
RUN dnf install -y --nogpgcheck "https://dl.fedoraproject.org/pub/epel/epel-release-latest-$(rpm -E %rhel).noarch.rpm" \
    "https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm" \
    "https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm"

# Upgrade all packages
RUN dnf upgrade -y

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN dnf install -y $(sh /tmp/pkg-validator.sh --pkgmgr dnf -- \
    bash-completion \
    bc \
    bzip2 \
    cracklib-dicts \
    curl \
    diffutils \
    dnf-plugins-core \
    findutils \
    glibc-all-langpacks \
    glibc-common \
    glibc-locale-source \
    gnupg2 \
    gnupg2-smime \
    hostname \
    iproute \
    iputils \
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
    sudo \
    tcpdump \
    time \
    systemd \
    traceroute \
    tree \
    tzdata \
    unzip \
    util-linux \
    util-linux-script \
    vte-profile \
    vulkan \
    wget \
    which \
    whois \
    words \
    xorg-x11-xauth \
    xz \
    zip \
    pipewire \
    pipewire-pulseaudio \
    wireplumber \
    ffmpeg \
    gstreamer1 \
    gstreamer1-plugins-base \
    gstreamer1-plugins-good \
    gstreamer1-plugins-bad-free \
    gstreamer1-plugins-bad-nonfree \
    gstreamer1-plugins-ugly \
    gstreamer1-plugin-libav \
    mesa-va-drivers-freeworld \
    mesa-vdpau-drivers-freeworld \
    libvpx \
    x264 \
    x265 \
    lame \
    opus)4

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN dnf clean all \
    && rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*