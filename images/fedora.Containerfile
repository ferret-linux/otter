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
    touch /usr/lib/otter/container.fedora

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || true

# Install rpm-fusion repos
RUN dnf install https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm \
    https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm \
    dnf5-plugins -y --refresh

# Enable Openh264 repos
RUN dnf config-manager setopt fedora-cisco-openh264.enabled=1

# Upgrade all packages
RUN dnf upgrade -y --refresh

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN dnf install -y $(sh /tmp/pkg-validator.sh --pkgmgr dnf -- \
    bash-completion \
    bc \
    bzip2 \
    cracklib-dicts \
    curl \
    diffutils \
    ffmpeg \
    mesa-va-drivers-freeworld \
    rpmfusion-free-release-tainted \
    rpmfusion-nonfree-release-tainted \
    mscore-fonts-all \
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
    traceroute \
    systemd \
    tree \
    tzdata \
    unzip \
    util-linux \
    util-linux-script \
    vte-profile \
    vulkan \
    wget \
    wget2-wget \
    which \
    whois \
    words \
    xorg-x11-xauth \
    xz \
    zip)

# Install nerd fonts
COPY images/scripts/nerd-fonts.sh /tmp/nerd-fonts.sh
RUN sh /tmp/nerd-fonts.sh

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