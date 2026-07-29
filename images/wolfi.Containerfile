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

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.wolfi

# Upgrade all packages
RUN apk update && apk upgrade

# Install dinit, built in the dinit-builder stage above (not packaged by Wolfi)
COPY --from=dinit-builder /dinit-out/usr/ /usr/

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apk add --force-overwrite $(sh /tmp/pkg-validator.sh --pkgmgr apk -- \
    bash \
    fish \
    zsh \
    bc \
    busybox \
    bzip2 \
    coreutils \
    curl \
    diffutils \
    ffmpeg \
    findmnt \
    findutils \
    gnupg \
    gnutar \
    gpg \
    gst-plugins-base \
    gst-plugins-bad \
    gst-plugins-ugly \
    gst-plugins-good \
    iproute2 \
    iputils \
    keyutils \
    less \
    libcap \
    man-db \
    mesa \
    mount \
    ncurses \
    ncurses-terminfo \
    net-tools \
    openssh-client \
    pigz \
    pinentry \
    pipewire \
    pipewire-jack \
    pipewire-pulse \
    posix-libc-utils \
    procps \
    python3 \
    rsync \
    script \
    shadow \
    sudo \
    tcpdump \
    tree \
    tzdata \
    umount \
    unzip \
    util-linux \
    util-linux-login \
    util-linux-misc \
    vpl-gpu-rt \
    vulkan-loader \
    wget \
    xauth \
    xdg-desktop-portal \
    xdg-user-dirs \
    xdg-utils \
    xz \
    zip)

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*