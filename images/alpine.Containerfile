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
RUN touch /usr/lib/otter/container.alpine

# Upgrade all packages, install alpine-base, install the complete
# package set, and perform Apk cleanup in the same layer.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk update && \
    apk upgrade && \
    apk add alpine-base && \
    apk update && \
    apk upgrade -Ua && \
    apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
        bc \
        xz \
        tar \
        zsh \
        gpg \
        zip \
        bash \
        fish \
        curl \
        docs \
        less \
        lsof \
        pigz \
        sudo \
        tree \
        vte3 \
        wget \
        which \
        bzip2 \
        gnupg \
        mount \
        rsync \
        unzip \
        xauth \
        ffmpeg \
        libcap \
        procps \
        shadow \
        mandoc \
        tzdata \
        umount \
        openrc \
        gcompat \
        findmnt \
        mesa-gl \
        iputils \
        ncurses \
        python3 \
        tcpdump \
        wayland \
        alsa-lib \
        mesa-egl \
        pipewire \
        iproute2 \
        keyutils \
        pinentry \
        xwayland \
        man-pages \
        coreutils \
        xdg-utils \
        diffutils \
        findutils \
        net-tools \
        fontconfig \
        libc-utils \
        vpl-gpu-rt \
        util-linux \
        musl-utils \
        openrc-init \
        alpine-base \
        xdg-user-dirs \
        pipewire-jack \
        vulkan-loader \
        pipewire-pulse \
        bash-completion \
        gst-plugins-bad \
        util-linux-misc \
        shared-mime-info \
        gst-plugins-base \
        gst-plugins-ugly \
        gst-plugins-good \
        ncurses-terminfo \
        util-linux-login \
        xdg-desktop-portal \
        hicolor-icon-theme \
        desktop-file-utils \
        openssh-client-default) && \
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

# Install GPU driver packages for Alpine. Alpine splits mesa-dri and
# mesa-vulkan into multiple arch/vendor-specific sub-packages, so we
# resolve them by prefix search here, the same way otter-init does
# via `apk search -q mesa-dri` / `apk search -q mesa-vulkan` at
# container-create time.
RUN apk update && \
    apk add --force-overwrite $(apk search -q mesa-dri) $(apk search -q mesa-vulkan) && \
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

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh