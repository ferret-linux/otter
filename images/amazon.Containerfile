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
RUN touch /usr/lib/otter/container.amazonlinux

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || true

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Refresh metadata, upgrade the system, install the requested
# packages, and clean DNF metadata/cache in the same layer.
# NOTE: unlike rhel-family, this does not install EPEL - Amazon
# Linux 2023 does not define the %rhel rpm macro and is not an
# officially EPEL-supported distro.
RUN dnf clean expire-cache && \
    dnf makecache --refresh && \
    dnf upgrade -y && \
    dnf install -y spal-release && \
    dnf install -y --skip-broken $(sh /tmp/pkg-validator.sh --pkgmgr dnf -- \
        bc \
        xz \
        tar \
        mtr \
        pam \
        zip \
        zsh \
        unar \
        bash \
        curl \
        fish \
        less \
        lsof \
        pigz \
        sudo \
        time \
        tree \
        wget \
        bzip2 \
        libXi \
        rsync \
        unzip \
        which \
        whois \
        words \
        libva \
        gnupg2 \
        libX11 \
        libxcb \
        man-db \
        passwd \
        tzdata \
        vulkan \
        iproute \
        iputils \
        libXext \
        ncurses \
        python3 \
        systemd \
        tcpdump \
        wayland \
        libvdpau \
        alsa-lib \
        hostname \
        keyutils \
        nss-mdns \
        pinentry \
        pipewire \
        diffutils \
        findutils \
        krb5-libs \
        libXfixes \
        libXrandr \
        man-pages \
        procps-ng \
        xdg-utils \
        fontconfig \
        gstreamer1 \
        libXcursor \
        libXdamage \
        libXrender \
        mesa-libGL \
        traceroute \
        util-linux \
        wget2-wget \
        ffmpeg-free \
        libXinerama \
        mesa-libEGL \
        mesa-libgbm \
        vte-profile \
        glibc-common \
        gnupg2-smime \
        libxkbcommon \
        shadow-utils \
        xdg-user-dirs \
        libXcomposite \
        pipewire-alsa \
        vulkan-loader \
        cracklib-dicts \
        xorg-x11-xauth \
        mesa-va-drivers \
        bash-completion \
        openssh-clients \
        dnf-plugins-core \
        libxkbcommon-x11 \
        mesa-dri-drivers \
        shared-mime-info \
        util-linux-script \
        xdg-desktop-portal \
        desktop-file-utils \
        hicolor-icon-theme \
        pipewire-gstreamer \
        mesa-vdpau-drivers \
        glibc-all-langpacks \
        glibc-locale-source \
        mesa-vulkan-drivers \
        pipewire-pulseaudio \
        gstreamer1-plugins-base \
        gstreamer1-plugins-good \
        xorg-x11-server-Xwayland \
        gstreamer1-plugins-bad-free \
        pipewire-jack-audio-connection-kit) && \
    dnf upgrade -y && \
    dnf autoremove -y && \
    dnf clean all && \
    rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

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