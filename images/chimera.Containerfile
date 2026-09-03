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
RUN touch /usr/lib/otter/container.chimera

# Upgrade all packages, enable Chimera's user repository, install
# bootstrap packages, install the complete package set, and perform
# Apk/filesystem cleanup before this layer is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk update && \
    apk upgrade && \
    apk add chimera-repo-user && \
    apk update && \
    apk add base-bootstrap && \
    apk upgrade -Ua && \
    apk update && \
    apk upgrade && \
    apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
        bc \
        xz \
        zsh \
        gpg \
        vte \
        zip \
        bash \
        fish \
        curl \
        gtar \
        less \
        lsof \
        pigz \
        sudo \
        tree \
        wget \
        which \
        bc-gh \
        bzip2 \
        gnupg \
        mount \
        rsync \
        dinit \
        unzip \
        wget2 \
        xauth \
        libva \
        ffmpeg \
        libcap \
        procps \
        shadow \
        tzdata \
        umount \
        findmnt \
        iputils \
        ncurses \
        openssh \
        python3 \
        tcpdump \
        wayland \
        pipewire \
        iproute2 \
        xwayland \
        keyutils \
        mesa-dri \
        opendoas \
        pinentry \
        coreutils \
        xdg-utils \
        diffutils \
        findutils \
        net-tools \
        vpl-gpu-rt \
        util-linux \
        mesa-vulkan \
        libcap-progs \
        ncurses-term \
        base-full-man \
        xdg-user-dirs \
        pipewire-jack \
        dinit-chimera \
        vulkan-loader \
        pipewire-alsa \
        pipewire-pulse \
        base-bootstrap \
        bash-completion \
        gst-plugins-bad \
        util-linux-misc \
        gst-plugins-base \
        gst-plugins-ugly \
        gst-plugins-good \
        libarchive-progs \
        ncurses-terminfo \
        util-linux-login \
        util-linux-mount \
        xdg-desktop-portal \
        pipewire-gstreamer) && \
    apk update && \
    apk upgrade && \
    apk cache clean && \
    rm -rf \
        /var/cache/apk/* \
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