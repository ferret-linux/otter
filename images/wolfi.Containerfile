# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE

# Build stage: compile dinit from source, since Wolfi does not package
# any init system (by design - see chainguard's own Wolfi FAQ).
# Built against the same base image to keep glibc ABI consistent with
# the final image the binaries will run in.
FROM ${IMAGE} AS dinit-builder
RUN apk update && apk add --no-cache build-base git curl m4
RUN DINIT_TAG="$(curl -fsSL https://api.github.com/repos/davmac314/dinit/releases/latest | grep '"tag_name"' | head -1 | cut -d'"' -f4)"; \
    if [ -z "${DINIT_TAG}" ]; then DINIT_TAG="v0.22.1"; fi; \
    echo "Building dinit ${DINIT_TAG}"; \
    git clone --branch "${DINIT_TAG}" --depth 1 https://github.com/davmac314/dinit /usr/src/dinit
WORKDIR /usr/src/dinit
RUN make mconfig && make -j$(nproc) && make DESTDIR=/dinit-out install

FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.wolfi

# Install dinit, built in the dinit-builder stage above (not packaged by Wolfi)
COPY --from=dinit-builder /dinit-out/usr/ /usr/

# Make dinit available as the system init
RUN ln -sf /usr/bin/dinit /sbin/init

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk update && \
    apk upgrade && \
    apk add wolfi-base && \
    apk update && \
    apk upgrade -Ua && \
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
        python3 \
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
        wolfi-base \
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

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh