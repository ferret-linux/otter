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

# CachyOS ships with a preconfigured pacman setup, including colored output,
# ILoveCandy, signature verification, package extraction defaults, and progress
# bar behavior. Keep the upstream configuration unchanged instead of overriding
# or duplicating it here.

# Identify the container as SteamOS while preserving the complete upstream
# CachyOS os-release metadata, including VERSION, BUILD_ID, HOME_URL, and
# other compatibility fields used by applications and system tooling.
RUN sed -i \
        -e 's/^NAME=.*/NAME="SteamOS"/' \
        -e 's/^PRETTY_NAME=.*/PRETTY_NAME="SteamOS OCI (cachyOSv3)"/' \
        /usr/lib/os-release

# Disable pacman's download sandbox for Podman container-build compatibility.
RUN sed -i '/^[[:space:]]*DisableSandbox[[:space:]]*$/d' /etc/pacman.conf \
    && sed -i '0,/^\[options\]/s/^\[options\]/[options]\nDisableSandbox/' /etc/pacman.conf

# CachyOS provides its own optimized repositories alongside the standard Arch
# repositories, including multilib. The gaming stack used by this image is
# sourced from these repositories, so Chaotic-AUR is intentionally not enabled
# to avoid introducing an additional third-party package repository.

# Upgrade all packages, install the complete package set, perform
# pacman cleanup, and remove filesystem caches before this layer
# is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN pacman -Syyu --noconfirm --needed && \
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
        vulkan-driver \
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
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install common 32-bit gaming dependencies from multilib for Steam, Wine, Proton, and games.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        lib32-xz \
        lib32-acl \
        lib32-gmp \
        lib32-icu \
        lib32-lz4 \
        lib32-nss \
        lib32-ogg \
        lib32-curl \
        lib32-dbus \
        lib32-flac \
        lib32-krb5 \
        lib32-mesa \
        lib32-nspr \
        lib32-opus \
        lib32-sdl2 \
        lib32-sdl3 \
        lib32-zlib \
        lib32-zstd \
        lib32-bzip2 \
        lib32-expat \
        lib32-glibc \
        lib32-libnm \
        lib32-libva \
        lib32-libxi \
        lib32-pcre2 \
        lib32-vkd3d \
        lib32-glib2 \
        lib32-brotli \
        lib32-giflib \
        lib32-gnutls \
        lib32-libcap \
        lib32-libdrm \
        lib32-libelf \
        lib32-libffi \
        lib32-libndp \
        lib32-libpng \
        lib32-libpsl \
        lib32-libx11 \
        lib32-libxau \
        lib32-libxcb \
        lib32-libxss \
        lib32-nettle \
        lib32-openal \
        lib32-libidn2 \
        lib32-libpcap \
        lib32-libthai \
        lib32-libwebp \
        lib32-libxext \
        lib32-libxml2 \
        lib32-libxtst \
        lib32-ncurses \
        lib32-openssl \
        lib32-p11-kit \
        lib32-systemd \
        lib32-wayland \
        lib32-libssh2 \
        lib32-alsa-lib \
        lib32-gcc-libs \
        lib32-libdecor \
        lib32-libglvnd \
        lib32-libinput \
        lib32-libproxy \
        lib32-libpulse \
        lib32-libtasn1 \
        lib32-pipewire \
        lib32-gamemode \
        lib32-mangohud \
        lib32-freetype2 \
        lib32-libgcrypt \
        lib32-libtheora \
        lib32-libunwind \
        lib32-libvorbis \
        lib32-libxcrypt \
        lib32-libxfixes \
        lib32-libxrandr \
        lib32-llvm-libs \
        lib32-libngtcp2 \
        lib32-fontconfig \
        lib32-jpeg-turbo \
        lib32-libasyncns \
        lib32-libxcursor \
        lib32-libxdamage \
        lib32-libxrender \
        lib32-lm_sensors \
        lib32-libnghttp2 \
        lib32-libnghttp3 \
        lib32-libxinerama \
        lib32-spirv-tools \
        lib32-alsa-plugins \
        lib32-libgpg-error \
        lib32-libpciaccess \
        lib32-libxshmfence \
        lib32-vulkan-intel \
        lib32-libxkbcommon \
        lib32-libxcomposite \
        lib32-vulkan-driver \
        lib32-vulkan-radeon \
        lib32-libdisplay-info \
        lib32-libxcrypt-compat \
        lib32-xcb-util-keysyms \
        lib32-libxkbcommon-x11 \
        lib32-vulkan-icd-loader \
        lib32-libsysprof-capture \
        lib32-vulkan-mesa-implicit-layers) && \
    pacman -Syy && \
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install common native gaming infrastructure and runtime helpers for
# Steam, Wine, Proton, Heroic, Lutris, and other game launchers.
# 32-bit gaming libraries are installed separately above.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        dxvk \
        p7zip \
        seatd \
        unrar \
        vkd3d \
        hwdata \
        openal \
        glslang \
        shaderc \
        gamemode \
        libdecor \
        libinput \
        mangohud \
        pciutils \
        usbutils \
        vkbasalt \
        gamescope \
        gst-libav \
        wine-mono \
        xorg-xset \
        cabextract \
        fluidsynth \
        mesa-demos \
        mesa-utils \
        noto-fonts \
        ttf-dejavu \
        wine-gecko \
        winetricks \
        xorg-xprop \
        libva-utils \
        spirv-tools \
        xcb-util-wm \
        xorg-xgamma \
        xorg-xinput \
        xorg-xrandr \
        vulkan-tools \
        pipewire-alsa \
        xorg-xsetroot \
        xorg-xwayland \
        gst-plugin-gtk \
        noto-fonts-cjk \
        ttf-liberation \
        vulkan-headers \
        xcb-util-cursor \
        xcb-util-errors \
        noto-fonts-emoji \
        noto-fonts-extra \
        xcb-util-keysyms \
        xorg-server-xvfb \
        vulkan-icd-loader \
        wayland-protocols \
        gst-plugin-pipewire \
        xcb-util-renderutil \
        power-profiles-daemon \
        xdg-desktop-portal-gtk) && \
    pacman -Syy && \
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install the Wine runtime and Wine-specific gaming components.
# Common graphics, audio, multimedia, input, and 32-bit dependencies
# are already installed above.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        wine \
        wine-mono \
        wine-gecko \
        wine-cachyos-opt) && \
    pacman -Syy && \
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install Proton compatibility tools and Proton utilities.
# Core Wine, Vulkan, DXVK, VKD3D, audio, fonts, and 32-bit dependencies
# are already installed above.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        proton \
        protonplus \
        protontricks \
        umu-launcher \
        proton-cachyos-slr) && \
    pacman -Syy && \
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
    rm -rf \
        /var/cache/pacman/pkg/* \
        /var/log/* \
        /var/tmp/*

# Install common gaming launchers, emulation, and Steam device support.
RUN pacman -S --needed --noconfirm $(sh /tmp/pkg-validator.sh --pkgmgr pacman -- \
        steam \
        lutris \
        bottles \
        retroarch \
        steam-devices \
        heroic-games-launcher) && \
    pacman -Syy && \
    pacman -Syu --noconfirm && \
    pacman -Scc --noconfirm && \
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

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

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