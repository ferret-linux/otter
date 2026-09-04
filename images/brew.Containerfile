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
# /usr/local -- that's macOS-only). Its bin directory is not added to
# the environment automatically, so otter bakes Homebrew's paths into
# the image environment. Commands executed through `su - linuxbrew`
# additionally initialize Homebrew with `brew shellenv`, since `su -`
# creates a fresh login environment.
ENV HOMEBREW_PREFIX=/home/linuxbrew/.linuxbrew
ENV HOMEBREW_CELLAR=/home/linuxbrew/.linuxbrew/Cellar
ENV HOMEBREW_REPOSITORY=/home/linuxbrew/.linuxbrew/Homebrew
ENV PATH="/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin:${PATH}"
ENV MANPATH="/home/linuxbrew/.linuxbrew/share/man:${MANPATH}"
ENV INFOPATH="/home/linuxbrew/.linuxbrew/share/info:${INFOPATH}"

# The image keeps root as its default user (like every other
# Containerfile) -- otter-init, not the image's USER, is what
# determines the live runtime user. brew itself never runs as root:
# the installer above switched to linuxbrew via `su -`, and the
# housekeeping below does exactly the same, since Homebrew on Linux
# is documented to misbehave when formulae are run/owned by root.
RUN su - linuxbrew -c ' \
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" && \
    brew doctor && \
    brew missing && \
    brew autoremove && \
    brew services cleanup && \
    brew cleanup --prune=all -s \
    '

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

######################################################################################

# Install Otter needed deps from APT/Debian Repos (copied from debian.Containerfile)
# Upgrade all packages, install the complete package set, perform
# Apt cleanup, and remove Apt/filesystem caches before this layer
# is committed.
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN apt-get update && \
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
        unrar \
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
        systemd \
        python3 \
        tcpdump \
        hostname \
        iproute2 \
        keyutils \
        pipewire \
        libopus0 \
        manpages \
        xz-utils \
        xwayland \
        xdg-utils \
        apt-utils \
        diffutils \
        findutils \
        libflac12 \
        libvdpau1 \
        libkrb5-3 \
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
        pipewire-jack \
        libvte-common \
        pipewire-alsa \
        pipewire-pulse \
        openssh-client \
        mesa-va-drivers \
        bash-completion \
        libgl1-mesa-dri \
        libgl1-mesa-glx \
        pinentry-curses \
        libavcodec-extra \
        libpipewire-0.3-0 \
        libnss-myhostname \
        libwayland-client0 \
        gstreamer1.0-tools \
        xdg-desktop-portal \
        gstreamer1.0-libav \
        mesa-vulkan-drivers \
        gstreamer1.0-pipewire \
        gstreamer1.0-plugins-bad \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        intel-media-va-driver-non-free) && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    apt-get clean && \
    rm -rf \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/tmp/*

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