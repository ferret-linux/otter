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
RUN touch /usr/lib/otter/container.steamos

# Configure pacman
RUN sed -i "s|NoExtract.*||g" /etc/pacman.conf \
    && sed -i "s|NoProgressBar.*||g" /etc/pacman.conf \
    && sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

# Set the container OS identity to SteamOS while retaining Arch compatibility.
RUN rm -f /etc/os-release /usr/lib/os-release \
    && touch /usr/lib/os-release \
    && ln -s /usr/lib/os-release /etc/os-release \
    && cat > /usr/lib/os-release <<'EOF'
NAME="SteamOS"
PRETTY_NAME="SteamOS OCI (otter)"
ID=steamos
ID_LIKE=arch
EOF

# Enable the multilib repository for Steam, gaming libraries, and 32-bit dependencies.
RUN if grep -q '^\s*\[multilib\]' /etc/pacman.conf; then \
        sed -i '/^\s*\[multilib\]/,/^\s*\[/ s/^\s*#\s*//' /etc/pacman.conf; \
    else \
        printf '\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n' >> /etc/pacman.conf; \
    fi

# Enable the Chaotic-AUR repository.
RUN pacman-key --recv-key 3056513887B78AEB --keyserver keyserver.ubuntu.com \
    && pacman-key --lsign-key 3056513887B78AEB \
    && pacman -U --noconfirm \
        'https://cdn-mirror.chaotic.cx/chaotic-aur/chaotic-keyring.pkg.tar.zst' \
        'https://cdn-mirror.chaotic.cx/chaotic-aur/chaotic-mirrorlist.pkg.tar.zst' \
    && printf '\n[chaotic-aur]\nInclude = /etc/pacman.d/chaotic-mirrorlist\n' >> /etc/pacman.conf

# Upgrade all packages, install the complete package set, perform
# pacman cleanup, and remove filesystem caches before this layer
# is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN pacman -Sy --noconfirm --needed archlinux-keyring && \
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
        systemd \
        iputils \
        mlocate \
        ncurses \
        openssh \
        tcpdump \
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
        vpl-gpu-rt \
        base-devel \
        traceroute \
        util-linux \
        vte-common \
        xorg-xauth \
        vulkan-intel \
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
        xdg-desktop-portal) && \
    pacman -Syy && \
    pacman -Syu && \
    pacman -Scc && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install common 32-bit gaming dependencies from multilib for Steam, Wine, Proton, and games.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        lib32-xz \
        lib32-icu \
        lib32-lz4 \
        lib32-nss \
        lib32-ogg \
        lib32-curl \
        lib32-dbus \
        lib32-flac \
        lib32-mesa \
        lib32-nspr \
        lib32-opus \
        lib32-sdl2 \
        lib32-sdl3 \
        lib32-zlib \
        lib32-bzip2 \
        lib32-expat \
        lib32-glibc \
        lib32-libxi \
        lib32-giflib \
        lib32-gnutls \
        lib32-libdrm \
        lib32-libogg \
        lib32-libpng \
        lib32-libx11 \
        lib32-openal \
        lib32-libwebp \
        lib32-libxext \
        lib32-libxml2 \
        lib32-systemd \
        lib32-wayland \
        lib32-alsa-lib \
        lib32-gcc-libs \
        lib32-libdecor \
        lib32-libglvnd \
        lib32-libpulse \
        lib32-pipewire \
        lib32-freetype2 \
        lib32-libtheora \
        lib32-libvorbis \
        lib32-libxfixes \
        lib32-fontconfig \
        lib32-jpeg-turbo \
        lib32-libasyncns \
        lib32-libxcursor \
        lib32-libxrender \
        lib32-libxinerama \
        lib32-alsa-plugins \
        lib32-libpciaccess \
        lib32-vulkan-intel \
        lib32-vulkan-radeon \
        lib32-vulkan-icd-loader) && \
    pacman -Syy && \
    pacman -Syu && \
    pacman -Scc && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh