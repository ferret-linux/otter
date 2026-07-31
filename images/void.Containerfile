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
    bc \
    xz \
    zsh \
    mtr \
    nss \
    zip \
    bash \
    curl \
    less \
    lsof \
    pigz \
    sudo \
    time \
    tree \
    vte3 \
    wget \
    x265 \
    opus \
    lame \
    bzip2 \
    rsync \
    runit \
    unzip \
    which \
    xauth \
    gnupg2 \
    man-db \
    shadow \
    tzdata \
    ffmpeg \
    libvpx \
    ncurses \
    openssh \
    python3 \
    libx264 \
    libflac \
    iproute2 \
    mesa-dri \
    mit-krb5 \
    pinentry \
    pipewire \
    diffutils \
    findutils \
    inetutils \
    procps-ng \
    gstreamer \
    fish-shell \
    traceroute \
    util-linux \
    wireplumber \
    pinentry-tty \
    mit-krb5-libs \
    vulkan-loader \
    pipewire-pulse \
    bash-completion \
    mit-krb5-client \
    gst-plugins-bad \
    gst-plugins-base \
    gst-plugins-good \
    gst-plugins-ugly \
    mesa-vulkan-intel \
    mesa-vulkan-radeon)

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