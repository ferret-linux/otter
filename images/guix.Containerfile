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

# No init runs during build, so guix-install.sh's own init detection
# resolved to NA and skipped enabling guix-daemon (its systemd branch
# is the only place that happens). Real systemd only starts later, at
# container runtime via otter-init's exec of /usr/lib/systemd/systemd
# as PID 1 -- by then guix-install.sh has already finished. Replicate
# its unit-enable step by hand instead, reading each unit's own
# WantedBy=/RequiredBy= line rather than hardcoding a target, so
# otter-init's generic systemd exec picks guix-daemon up on its own.
# configure_substitute_discovery (upstream's own function, prompts
# "enable local-network substitute discovery?") is skipped here on
# purpose: it only fires inside a live interactive prompt, and since
# this whole install runs non-interactively there's no answer to
# replicate -- omitting it matches its own "no" branch, i.e. the
# --discover=no guix ships by default is left untouched.
RUN unit_src=/root/.config/guix/current/lib/systemd/system && \
    for unit in guix-daemon.service gnu-store.mount; do \
        cp "${unit_src}/${unit}" "/etc/systemd/system/${unit}" && \
        chmod 664 "/etc/systemd/system/${unit}" && \
        for target in $(sed -n 's/^WantedBy=//p' "${unit_src}/${unit}"); do \
            mkdir -p "/etc/systemd/system/${target}.wants" && \
            ln -sf "../${unit}" "/etc/systemd/system/${target}.wants/${unit}"; \
        done; \
        for target in $(sed -n 's/^RequiredBy=//p' "${unit_src}/${unit}"); do \
            mkdir -p "/etc/systemd/system/${target}.requires" && \
            ln -sf "../${unit}" "/etc/systemd/system/${target}.requires/${unit}"; \
        done; \
    done

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