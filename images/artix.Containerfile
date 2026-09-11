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
RUN touch /usr/lib/otter/container.artixlinux

# Configure pacman
RUN sed -i "s|NoExtract.*||g" /etc/pacman.conf \
    && sed -i "s|NoProgressBar.*||g" /etc/pacman.conf \
    && sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

# Upgrade all packages, install the complete package set, perform
# pacman cleanup, and remove filesystem caches before this layer
# is committed.
# NOTE: unlike archlinux, this does not install/require systemd -
# Artix does not ship it, and otter's --init flag is systemd-only
# (see pkg/containermanager/providers/docker.go), so --init is not
# supported on this image regardless.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN pacman -Sy --noconfirm --needed artix-keyring && \
    pacman -Syyu --noconfirm --needed && \
    pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        bc \
        zsh \
        git \
        tar \
        mtr \
        zip \
        fish \
        bash \
        curl \
        less \
        lsof \
        mesa \
        pigz \
        sudo \
        time \
        tree \
        wget \
        base \
        libva \
        glibc \
        gnupg \
        rsync \
        unzip \
        words \
        ffmpeg \
        man-db \
        python \
        shadow \
        tzdata \
        openrc \
        iputils \
        mlocate \
        ncurses \
        openssh \
        tcpdump \
        wayland \
        pipewire \
        fakeroot \
        keyutils \
        nss-mdns \
        pinentry \
        xdg-utils \
        diffutils \
        findutils \
        inetutils \
        man-pages \
        procps-ng \
        base-devel \
        traceroute \
        util-linux \
        vte-common \
        vpl-gpu-rt \
        xorg-xauth \
        vulkan-intel \
        xorg-xwayland \
        pipewire-alsa \
        pipewire-jack \
        xdg-user-dirs \
        vulkan-radeon \
        pipewire-pulse \
        bash-completion \
        gst-plugins-bad \
        util-linux-libs \
        gst-plugins-base \
        gst-plugins-good \
        gst-plugins-ugly \
        xdg-desktop-portal \
        gst-plugin-pipewire) && \
    pacman -Syy && \
    pacman -Syu && \
    pacman -Scc && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable
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

# Ship the default otter starship integration config.
COPY images/scripts/starship.toml /usr/lib/otter/helpers/starship.toml