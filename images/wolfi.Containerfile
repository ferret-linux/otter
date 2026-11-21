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
RUN touch /usr/lib/otter/container.wolfi

# Install OpenRC from Wolfi's own repositories and wire up /sbin/init.
# Otter's init integration detects OpenRC via /usr/sbin/openrc and execs
# /usr/sbin/openrc-init when it is PID 1.
RUN apk add --no-cache openrc \
    && ln -sf /usr/sbin/openrc-init /sbin/init

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk update && \
    apk upgrade && \
    apk add wolfi-base && \
    apk update && \
    apk upgrade -Ua && \
    PYTHON_PKG="$(apk search python 2>/dev/null | grep -E '^python-3\.[0-9]+-[0-9]' | cut -d- -f1,2 | sort -Vu | tail -n1)"; \
    test -n "${PYTHON_PKG}" || { echo "ERROR: no python-3.X package found in Wolfi repos" >&2; exit 1; }; \
    apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
        bc \
        xz \
        zsh \
        gpg \
        zip \
        bash \
        fish \
        curl \
        less \
        mesa \
        pigz \
        sudo \
        tree \
        wget \
        which \
        bzip2 \
        gnupg \
        mount \
        rsync \
        unzip \
        xauth \
        ffmpeg \
        gnutar \
        libcap \
        man-db \
        procps \
        script \
        shadow \
        tzdata \
        umount \
        libdrm \
        busybox \
        findmnt \
        iputils \
        ncurses \
        ${PYTHON_PKG} \
        tcpdump \
        wayland \
        xwayland \
        iproute2 \
        keyutils \
        pinentry \
        pipewire \
        alsa-lib \
        gstreamer \
        coreutils \
        diffutils \
        findutils \
        net-tools \
        xdg-utils \
        gst-libav \
        alsa-utils \
        util-linux \
        vpl-gpu-rt \
        wireplumber \
        pipewire-jack \
        vulkan-loader \
        xdg-user-dirs \
        openssh-client \
        pipewire-pulse \
        bash-completion \
        gst-plugins-bad \
        util-linux-misc \
        gst-plugins-base \
        gst-plugins-ugly \
        gst-plugins-good \
        ncurses-terminfo \
        posix-libc-utils \
        util-linux-login \
        gstreamer-pipewire \
        xdg-desktop-portal) && \
    apk update && \
    apk upgrade && \
    apk cache clean && \
    rm -rf \
        /var/cache/apk/* \
        /var/log/* \
        /var/tmp/*

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Safety net: ensure a python3 binary is resolvable. Wolfi's versioned
# python-3.X packages normally provide /usr/bin/python3 already.
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