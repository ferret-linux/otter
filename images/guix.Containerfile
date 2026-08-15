# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

# No hardcoded base -- IMAGE is supplied by the build pipeline, same
# convention as nix.Containerfile and brew.Containerfile. Expected
# value: debian:stable-slim (floating stable pointer, not pinned).
ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# Guix's install script needs gpg to verify the release signature,
# and wget/tar/xz to fetch and unpack the substitute binary tarball
# it installs itself from. sudo is needed because the installer sets
# up /gnu/store and the build-users-group as root, even though guix
# itself is invoked as a normal user afterward.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apt-get update && \
    apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        ca-certificates \
        gnupg \
        wget \
        tar \
        git \
        make \
        zstd \
        gcc \
        curl \
        systemd \
        xz-utils \
        procps \
        sudo) \
    && rm -rf /var/lib/apt/lists/*

RUN rm -rf /usr/lib/os-release /etc/os-release && \
    printf 'ID=guix\nID_LIKE=debian\nNAME="GUIX (debian)"\nPRETTY_NAME="GNU/GUIX (debian)"\n' > /etc/os-release

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh
RUN mkdir -p /usr/lib/otter && touch /usr/lib/otter/container.guix

# guix-install.sh handles all of this itself when run non-
# interactively (build-users-group creation, /gnu/store ownership,
# guix-daemon service files) -- called with the upstream-documented
# non-interactive flags rather than reimplementing that setup by
# hand, so otter tracks whatever the installer's own logic does
# release to release instead of drifting from it.
RUN cd /tmp && \
    wget -q https://guix.gnu.org/install.sh -O guix-install.sh && \
    chmod +x guix-install.sh && \
    GUIX_ALLOW_OVERWRITE=yes sh -c 'yes "" | ./guix-install.sh' && \
    rm -f guix-install.sh

# Container builds can't rely on systemd to supervise guix-daemon the
# way a real Guix System install would (unit files are dropped by
# the installer above but nothing here starts them -- this image has
# no init running at build time). guix commands that build/substitute
# packages need guix-daemon reachable; flagging this as the same
# category of unverified/best-effort area as the Nix image's GL
# driver dispatch note: otter-init is expected to actually start
# guix-daemon at container runtime (e.g. as a supervised service)
# rather than it being handled here at build time.
ENV GUIX_LOCPATH=/root/.guix-profile/lib/locale
ENV PATH="/root/.config/guix/current/bin:/root/.guix-profile/bin:/root/.guix-profile/sbin:${PATH}"

# Pin the "guix pull" channel implicitly by NOT running `guix pull`
# here -- doing so would fetch and build the latest Guix from source
# at image-build time, which is slow and non-deterministic across
# rebuilds. The freshly-installed guix-install.sh binary is used as-
# is; users who want a newer Guix can `guix pull` themselves at
# runtime once guix-daemon is up, same spirit as the Nix image
# leaving `nix flake update` to the user rather than baking it in.

# Housekeeping: same GC posture as the Nix image -- collect garbage
# from the install process itself so the image doesn't ship build
# artifacts nobody asked for, without needing guix-daemon running
# yet (guix gc operates on the store directly).
RUN guix gc -d 2>/dev/null || true

RUN rm -rf /var/log/* /var/tmp/*