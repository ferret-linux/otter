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

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Enable non-free and contrib repos
RUN sed -i 's/Components: main/Components: main contrib non-free non-free-firmware/' /etc/apt/sources.list.d/debian.sources

# Guix's install script needs gpg to verify the release signature,
# and wget/tar/xz to fetch and unpack the substitute binary tarball
# it installs itself from. sudo is needed because the installer sets
# up /gnu/store and the build-users-group as root, even though guix
# itself is invoked as a normal user afterward.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apt-get update && \
    apt-get upgrade -y && \
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
        python3 \
        locales \
        xz-utils \
        procps \
        sudo) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/tmp/*

# Locale setup
RUN sed -i "s|# en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

RUN rm -rf /usr/lib/os-release /etc/os-release && \
    printf 'ID=guix\nID_LIKE=debian\nNAME="GUIX (debian)"\nPRETTY_NAME="GNU/GUIX (debian)"\n' > /etc/os-release

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh
RUN mkdir -p /usr/lib/otter && touch /usr/lib/otter/container.guix

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

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

# install_unprivileged_daemon() (called from sys_create_build_user)
# checks `[ "$INIT_SYS" = systemd ]` to decide whether guix-daemon
# should run as the unprivileged "guix-daemon" user. With NA at build
# time it took the else branch instead: guixbuilder01-10 build users
# were created and /gnu, /var/guix were left root:root from the raw
# tar extract. But the real guix-daemon.service (verified:
# `User=guix-daemon`, `AmbientCapabilities=CAP_CHOWN`) expects the
# unprivileged setup regardless of which branch ran at build time --
# without this, the daemon has no user to run as and no ownership of
# the store it needs at container runtime. Replicated by hand below,
# matching install_unprivileged_daemon()'s branch and create_account().
RUN if getent group kvm > /dev/null; then kvmgroup=",kvm"; else kvmgroup=""; fi && \
    groupadd --system guix-daemon && \
    useradd --system -g guix-daemon -G "guix-daemon${kvmgroup}" \
        -d /var/empty -s "$(command -v nologin)" \
        -c "Unprivileged Guix Daemon User" guix-daemon && \
    if getent group kvm > /dev/null; then \
        kvmgid="$(getent group kvm | cut -f3 -d:)" && \
        echo "guix-daemon:${kvmgid}:1" >> /etc/subgid; \
    fi && \
    chown -R guix-daemon:guix-daemon /gnu /var/guix && \
    chown -R root:root /var/guix/profiles/per-user/root && \
    mkdir -p /var/log/guix && \
    chown guix-daemon:guix-daemon /var/log/guix && \
    chmod 755 /var/log/guix

# guixbuilder01-10 and guixbuild above are the privileged-daemon build
# users guix-install.sh creates on the NA/non-systemd path -- mutually
# exclusive with the unprivileged setup just applied (upstream's own
# install_unprivileged_daemon() if/else never creates both). Removed
# so the image doesn't ship unused accounts alongside guix-daemon.
RUN for i in $(seq -w 1 10); do userdel -f "guixbuilder${i}" 2>/dev/null || true; done && \
    groupdel guixbuild 2>/dev/null || true

# guix-install.sh's own init detection also resolved to NA, so
# sys_enable_guix_daemon() never installed/enabled the systemd units
# either (its systemd branch is the only place that happens). Real
# systemd only starts later, at container runtime via otter-init's
# exec of /usr/lib/systemd/systemd as PID 1 -- by then guix-install.sh
# has already finished. Replicated by hand: copy the units guix
# already shipped, reading each unit's own WantedBy=/RequiredBy= line
# rather than hardcoding a target, so otter-init's generic systemd
# exec picks guix-daemon up on its own once systemd is actually
# running. configure_substitute_discovery (upstream's own function,
# prompts "enable local-network substitute discovery?") is skipped on
# purpose -- it only fires inside a live interactive prompt, and since
# this whole install runs non-interactively there's no answer to
# replicate; omitting it leaves the --discover=no guix ships by
# default untouched, which is also the safer default for a container
# whose network namespace isn't a trusted LAN.
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

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh