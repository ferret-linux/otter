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
RUN touch /usr/lib/otter/container.void

# Configure xbps to not overwrite host-managed files
RUN mkdir -p /etc/xbps.d \
    && echo -e "noextract=/etc/passwd\nnoextract=/etc/hosts\nnoextract=/etc/host.conf\nnoextract=/etc/hostname\nnoextract=/etc/localtime\nnoextract=/etc/machine-id\nnoextract=/etc/resolv.conf" \
    > /etc/xbps.d/otter-ignore.conf

# Update xbps itself first
RUN xbps-install -Syu xbps

# Enable non-free repos
RUN xbps-install -Sy void-repo-nonfree \
    && xbps-install -Sy

# Upgrade all packages
RUN xbps-install -Syu
# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN xbps-install -Sy $(sh /tmp/pkg-validator.sh --pkgmgr xbps -- \
    bash-completion \
    bc \
    fish-shell \
    bash \
    zsh \
    bzip2 \
    curl \
    diffutils \
    findutils \
    gnupg2 \
    inetutils \
    iproute2 \
    less \
    lsof \
    man-db \
    mesa-dri \
    mesa-vulkan-intel \
    mesa-vulkan-radeon \
    mit-krb5 \
    mit-krb5-client \
    mit-krb5-libs \
    mtr \
    ncurses \
    nss \
    openssh \
    pigz \
    pinentry \
    pinentry-tty \
    procps-ng \
    python3 \
    rsync \
    runit \
    shadow \
    sudo \
    time \
    traceroute \
    tree \
    tzdata \
    unzip \
    util-linux \
    vulkan-loader \
    vte3 \
    wget \
    which \
    xauth \
    xz \
    zip \
    ffmpeg \
    gstreamer \
    gst-plugins-base \
    gst-plugins-good \
    gst-plugins-bad \
    gst-plugins-ugly \
    libx264 \
    x265 \
    libvpx \
    opus \
    libflac \
    lame \
    pipewire \
    pipewire-pulse \
    wireplumber)

# Locale setup (glibc only, musl does not use libc-locales)
RUN if [ -f /etc/default/libc-locales ]; then \
    sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/default/libc-locales \
    && xbps-reconfigure --force glibc-locales; \
    fi

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/xbps/* \
    /var/log/* \
    /var/tmp/*