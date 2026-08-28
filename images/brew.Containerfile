# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

# No hardcoded base -- IMAGE is supplied by the build pipeline, same
# convention as nix.Containerfile. Expected value: debian:stable-slim
# (Debian's floating stable pointer, not a pinned point release --
# rides forward automatically the same way nixpkgs-unstable does for
# the Nix image). Homebrew's own official homebrew/brew image is
# Ubuntu-based, but Debian slim satisfies the same actual
# requirement (glibc, bash, a real $HOME) with a smaller, more
# predictable base -- see project notes for the full rationale.
ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Enable non-free and contrib repos
RUN sed -i 's/Components: main/Components: main contrib non-free non-free-firmware/' /etc/apt/sources.list.d/debian.sources

# Homebrew's install script hard-requires these and none of them are
# present on debian:stable-slim by default. build-essential is
# needed because several common formulae/bottled fallbacks still
# compile from source on Linux when a bottle isn't available for the
# host's glibc version. procps is required by the installer itself
# (it shells out to `ps`).
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-suggests $(sh /tmp/pkg-validator.sh --pkgmgr apt -- \
        ca-certificates \
        curl \
        gcc \
        zstd \
        systemd \
        tar \
        wget \
        locales \
        python3 \
        git \
        build-essential \
        procps \
        file \
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
    printf 'ID=brew\nID_LIKE=debian\nNAME="Homebrew (debian)"\nPRETTY_NAME="Homebrew Linux (debian)"\n' > /etc/os-release

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Homebrew refuses to run as root and expects a normal user account
# with passwordless sudo (the installer itself needs sudo once, to
# create /home/linuxbrew/.linuxbrew, then drops privileges for
# everything after). "linuxbrew" is the conventional username the
# installer and Homebrew's own docs assume; keeping it avoids
# surprises in any brew-adjacent tooling that hardcodes that path.
RUN useradd -m -s /bin/bash linuxbrew && \
    echo "linuxbrew ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/linuxbrew && \
    chmod 0440 /etc/sudoers.d/linuxbrew

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.nix.brew 2>/dev/null || mkdir -p /usr/lib/otter && touch /usr/lib/otter/container.brew

# Run the official installer as the linuxbrew user. NONINTERACTIVE=1
# skips the confirmation prompt (otherwise the install hangs forever
# in a non-tty build) and makes have_sudo_access() call `sudo -n`
# (fail instead of prompt), which only succeeds because linuxbrew
# already has passwordless sudo configured above. CI is intentionally
# not set: in the real installer, CI only matters as an alternate way
# to derive NONINTERACTIVE=1 when it isn't already set directly --
# with NONINTERACTIVE=1 already present, CI is never read for
# anything else in the script, so setting it here would be dead
# weight left over from an earlier, inaccurate assumption.
RUN su - linuxbrew -c ' \
    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" \
    '

# Homebrew installs to /home/linuxbrew/.linuxbrew on Linux (not
# /usr/local -- that's macOS-only). Nothing puts its bin dir on
# $PATH by default outside of an interactive shell sourcing
# shellenv, so otter needs this baked in at the image level the same
# way the Nix image relies on its base's ONBUILD ENV for the profile
# bin dir.
ENV HOMEBREW_PREFIX=/home/linuxbrew/.linuxbrew
ENV HOMEBREW_CELLAR=/home/linuxbrew/.linuxbrew/Cellar
ENV HOMEBREW_REPOSITORY=/home/linuxbrew/.linuxbrew/Homebrew
ENV PATH="/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin:${PATH}"
ENV MANPATH="/home/linuxbrew/.linuxbrew/share/man:${MANPATH}"
ENV INFOPATH="/home/linuxbrew/.linuxbrew/share/info:${INFOPATH}"

# Homebrew on Linux is documented as expecting to run as a non-root
# user permanently (not just during install) -- formulae installed
# as root can produce permission errors on later `brew` invocations.
# Flagging as unverified rather than solved: otter-init's default
# user model may need to route into the linuxbrew user specifically
# for any brew-invoking workflow, same open-question flavor as the
# GL driver dispatch note in the Nix image.
USER linuxbrew
WORKDIR /home/linuxbrew

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Housekeeping: Homebrew's own installer leaves a git checkout and
# cache behind under HOMEBREW_REPOSITORY; brew cleanup trims
# formula caches/old versions without removing the install itself.
RUN brew doctor && \
    brew missing && \
    brew autoremove && \
    brew services cleanup && \
    brew cleanup --prune=all -s