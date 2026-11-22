# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}
ARG TAG

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.rhel-family

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || \
    sed -i '/tsflags=nodocs/d' /etc/yum.conf 2>/dev/null || true

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# EPEL is enabled for every tag. RPM Fusion (including its nonfree repo)
# is added only for nonfree builds; negativo17 builds add the upstream
# EPEL negativo17 repo instead.
RUN set -eux; \
    rpm --import "https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-$(rpm -E %rhel)"; \
    dnf install -y "https://dl.fedoraproject.org/pub/epel/epel-release-latest-$(rpm -E %rhel).noarch.rpm"; \
    case "${TAG:-}" in \
        nonfree) \
            dnf install -y \
                "https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm" \
                "https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm"; \
            ;; \
        negativo17) \
            command -v curl >/dev/null 2>&1 || dnf install -y curl; \
            curl -fsSLo /etc/yum.repos.d/negativo17.repo \
                https://negativo17.org/repos/epel-negativo17.repo; \
            rpm --import https://negativo17.org/repos/RPM-GPG-KEY-slaanesh; \
            ;; \
    esac; \
    dnf upgrade -y && \
    dnf autoremove -y && \
    dnf clean all && \
    rm -rf \
        /var/cache/dnf/* \
        /var/log/* \
        /var/tmp/*

# Upgrade the system, install the requested packages, and clean
# DNF metadata/cache in the same layer.
RUN set -eux; \
    case "${TAG:-}" in \
        nonfree) extra_packages="ffmpeg ffmpeg-libs mesa-freeworld libheif-freeworld libavcodec-freeworld xpra-codecs-freeworld gstreamer1-plugins-ugly mesa-va-drivers-freeworld mesa-vulkan-drivers-freeworld gstreamer1-plugins-bad-freeworld libaacs fdk-aac pipewire-codec-aptx" ;; \
        negativo17) extra_packages="ffmpeg ffmpeg-libs libaacs fdk-aac gstreamer1-plugins-ugly pipewire-codec-aptx mesa-libva libvdpau-va-gl" ;; \
        *) extra_packages="" ;; \
    esac; \
    dnf upgrade -y && \
    dnf install -y --allowerasing --skip-broken $(sh /tmp/pkg-validator.sh --pkgmgr dnf -- \
        bc \
        xz \
        zsh \
        mtr \
        pam \
        zip \
        fish \
        bash \
        curl \
        less \
        lsof \
        pigz \
        sudo \
        time \
        tree \
        wget \
        bzip2 \
        rsync \
        unzip \
        which \
        whois \
        words \
        gnupg2 \
        man-db \
        passwd \
        tzdata \
        vulkan \
        iproute \
        iputils \
        ncurses \
        python3 \
        tcpdump \
        systemd \
        pipewire \
        hostname \
        keyutils \
        nss-mdns \
        pinentry \
        xdg-utils \
        diffutils \
        findutils \
        krb5-libs \
        man-pages \
        procps-ng \
        traceroute \
        util-linux \
        wget2-wget \
        vte-profile \
        glibc-common \
        gnupg2-smime \
        shadow-utils \
        xdg-user-dirs \
        pipewire-alsa \
        cracklib-dicts \
        xorg-x11-xauth \
        bash-completion \
        openssh-clients \
        dnf-plugins-core \
        mesa-dri-drivers \
        util-linux-script \
        xdg-desktop-portal \
        pipewire-gstreamer \
        pipewire-pulseaudio \
        glibc-all-langpacks \
        glibc-locale-source \
        gstreamer1-plugins-base \
        gstreamer1-plugins-good \
        xorg-x11-server-Xwayland \
        pipewire-jack-audio-connection-kit \
        ${extra_packages}); \
    dnf upgrade -y && \
    dnf autoremove -y && \
    dnf clean all && \
    rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Fix Zsh zlogout on Sh/Enter
RUN rm -rf /etc/zlogout && touch /etc/zlogout

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