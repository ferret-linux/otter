# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}
ARG TAG

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.devuan

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Nonfree builds enable contrib/non-free/non-free-firmware before any apt
# install; free builds leave the base image's sources untouched.
RUN case "${TAG:-}" in \
        *-nonfree) \
            sed -i -E '/^deb /s/[[:space:]]+(main|contrib|non-free|non-free-firmware)([[:space:]]+(main|contrib|non-free|non-free-firmware))*[[:space:]]*$/ main contrib non-free non-free-firmware/' /etc/apt/sources.list; \
            ;; \
    esac

# Upgrade all packages, install the complete package set, perform
# Apt cleanup, and remove Apt/filesystem caches before this layer
# is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN set -eux; \
    case "${TAG:-}" in \
        *-nonfree) extra_packages="unrar intel-media-va-driver-non-free" ;; \
        *) extra_packages="" ;; \
    esac; \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        bc \
        zsh \
        mtr \
        zip \
        curl \
        fish \
        bash \
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
        unzip \
        xauth \
        dialog \
        gnupg2 \
        ffmpeg \
        libgl1 \
        man-db \
        passwd \
        procps \
        tzdata \
        libvpx9 \
        libegl1 \
        locales \
        python3 \
        tcpdump \
        hostname \
        iproute2 \
        keyutils \
        sysvinit \
        pipewire \
        libopus0 \
        manpages \
        xz-utils \
        xwayland \
        apt-utils \
        diffutils \
        findutils \
        libflac12 \
        libvdpau1 \
        libkrb5-3 \
        xdg-utils \
        libvulkan1 \
        traceroute \
        util-linux \
        libcap2-bin \
        wireplumber \
        libx264-164 \
        libx265-199 \
        libmp3lame0 \
        libnss-mdns \
        iputils-ping \
        libegl-mesa0 \
        libglx-mesa0 \
        ncurses-base \
        xdg-user-dirs \
        libvte-common \
        pipewire-alsa \
        pipewire-jack \
        pipewire-pulse \
        openssh-client \
        bash-completion \
        libgl1-mesa-dri \
        libgl1-mesa-glx \
        pinentry-curses \
        mesa-va-drivers \
        libavcodec-extra \
        libpipewire-0.3-0 \
        libnss-myhostname \
        xdg-desktop-portal \
        gstreamer1.0-tools \
        gstreamer1.0-libav \
        mesa-vulkan-drivers \
        gstreamer1.0-pipewire \
        gstreamer1.0-plugins-bad \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        ${extra_packages}) && \
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

# Ship the default otter starship integration config.
COPY images/scripts/starship.toml /usr/lib/otter/helpers/starship.toml