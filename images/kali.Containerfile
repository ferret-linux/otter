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
RUN touch /usr/lib/otter/container.kali

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Upgrade the system, install the requested packages, and perform
# Apt cleanup in the same layer.
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        bc \
        mtr \
        zip \
        zsh \
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
        gnupg \
        gpgsm \
        rsync \
        unrar \
        unzip \
        xauth \
        dialog \
        ffmpeg \
        gnupg2 \
        libgl1 \
        libva2 \
        libxi6 \
        man-db \
        passwd \
        procps \
        tzdata \
        libdrm2 \
        libegl1 \
        libvpx9 \
        libxcb1 \
        locales \
        python3 \
        systemd \
        tcpdump \
        hostname \
        iproute2 \
        keyutils \
        libopus0 \
        libx11-6 \
        libxext6 \
        manpages \
        pipewire \
        xwayland \
        xz-utils \
        apt-utils \
        diffutils \
        findutils \
        libflac12 \
        libkrb5-3 \
        xdg-utils \
        libvdpau1 \
        libva-drm2 \
        libvulkan1 \
        libxfixes3 \
        libxrandr2 \
        traceroute \
        util-linux \
        libcap2-bin \
        libmp3lame0 \
        libnss-mdns \
        libx264-164 \
        libx265-199 \
        libxcursor1 \
        libxdamage1 \
        libxrender1 \
        wireplumber \
        iputils-ping \
        libegl-mesa0 \
        libglx-mesa0 \
        libxinerama1 \
        ncurses-base \
        xdg-user-dirs \
        pipewire-alsa \
        pipewire-jack \
        libxkbcommon0 \
        libxcomposite1 \
        openssh-client \
        pipewire-pulse \
        mesa-va-drivers \
        bash-completion \
        libgl1-mesa-dri \
        libgl1-mesa-glx \
        libwayland-egl1 \
        pinentry-curses \
        libavcodec-extra \
        libnss-myhostname \
        libpipewire-0.3-0 \
        wayland-protocols \
        gstreamer1.0-libav \
        gstreamer1.0-tools \
        libwayland-client0 \
        libwayland-cursor0 \
        libwayland-server0 \
        libxkbcommon-x11-0 \
        xdg-desktop-portal \
        mesa-vulkan-drivers \
        gstreamer1.0-plugins-bad \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        intel-media-va-driver-non-free) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean \
    && rm -rf \
    /var/lib/apt/lists/* \
    /var/log/* \
    /var/tmp/*

# Install Kali's own curated tool tiers: core system essentials,
# the top-10 most commonly expected pentesting tools, and the
# headless (GUI-free) default toolset.
RUN apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        kali-linux-core \
        kali-tools-top10 \
        kali-linux-headless) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
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
RUN sed -i "s|# en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

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