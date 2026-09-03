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
RUN touch /usr/lib/otter/container.void

# Configure xbps to not overwrite host-managed files
RUN mkdir -p /etc/xbps.d \
    && echo -e "noextract=/etc/passwd\nnoextract=/etc/hosts\nnoextract=/etc/host.conf\nnoextract=/etc/hostname\nnoextract=/etc/localtime\nnoextract=/etc/machine-id\nnoextract=/etc/resolv.conf" \
    > /etc/xbps.d/otter-ignore.conf

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Update xbps, enable non-free repositories, upgrade the system,
# install the requested packages, and clean XBPS metadata/cache
# in the same layer.
RUN xbps-install -Syu xbps && \
    xbps-install -Sy void-repo-nonfree && \
    xbps-install -Sy && \
    xbps-install -Syu && \
    xbps-install -Sy $(sh /tmp/pkg-validator.sh --pkgmgr xbps -- \
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
        libdrm \
        gnupg2 \
        man-db \
        shadow \
        tzdata \
        ffmpeg \
        libvpx \
        wayland \
        ncurses \
        openssh \
        python3 \
        libx264 \
        libflac \
        alsa-lib \
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
        xdg-utils \
        gst-libav \
        alsa-utils \
        fish-shell \
        traceroute \
        util-linux \
        wireplumber \
        pinentry-tty \
        mit-krb5-libs \
        vulkan-loader \
        xdg-user-dirs \
        pipewire-pulse \
        bash-completion \
        mit-krb5-client \
        gst-plugins-bad \
        gst-plugins-base \
        gst-plugins-good \
        gst-plugins-ugly \
        mesa-vulkan-intel \
        mesa-vulkan-radeon \
        xdg-desktop-portal \
        gstreamer1-pipewire \
        xorg-server-xwayland) && \
    xbps-install -Su && \
    xbps-remove -O && \
    xbps-remove -o && \
    rm -rf \
    /var/cache/xbps/* \
    /var/log/* \
    /var/tmp/*

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

# Locale setup (glibc only, musl does not use libc-locales)
RUN if [ -f /etc/default/libc-locales ]; then \
    sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/default/libc-locales \
    && xbps-reconfigure --force glibc-locales; \
    fi

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Static-binary backstop for fish/nushell where the distro's own package
# manager doesn't carry them (checked via command -v inside the script).
COPY images/scripts/install-shell.sh /tmp/install-shell.sh
RUN sh /tmp/install-shell.sh fish nu

# Silence fish/nushell startup & welcome messages (see silence-shell.sh).
COPY images/scripts/silence-shell.sh /tmp/silence-shell.sh
RUN sh /tmp/silence-shell.sh

# Install starship (static musl binary, amd64/arm64) into
# /usr/local/bin. Official images only.
COPY images/scripts/install-starship.sh /tmp/install-starship.sh
RUN sh /tmp/install-starship.sh

# Ship the default otter starship integration, read via $STARSHIP_CONFIG set below.
# Users can override it any time with their own $STARSHIP_CONFIG or ~/.config/starship.toml.
COPY images/scripts/starship.toml /usr/lib/otter/helpers/starship.toml
ENV STARSHIP_CONFIG=/usr/lib/otter/helpers/starship.toml