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
RUN touch /usr/lib/otter/container.opensuse

# Configure zypper to install recommended packages and docs
RUN mkdir -p /etc/zypp/zypp.conf.d \
    && echo -e "[main]\nsolver.onlyRequires = false\nrpm.install.excludedocs = no" \
    > /etc/zypp/zypp.conf.d/99-otter.conf

# Lock parallel-printer-support as it can't be installed in rootless containers
RUN zypper al parallel-printer-support

# Enable the full Packman repo and VideoLAN for nonfree builds. Packman
# supplies the patent-encumbered codecs, libraries and media tools that the
# default repos omit; VideoLAN is the only source of libdvdcss2. URLs are
# derived from /etc/os-release so the dynamic base tags keep working without
# pinning Leap or Tumbleweed. VideoLAN gets a lower priority so Packman wins
# for packages both repos provide.
RUN set -eu; \
    case "${TAG:-}" in \
        *-nonfree) \
            . /etc/os-release; \
            case "${ID}" in \
                opensuse-leap) \
                    packman_url="https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Leap_${VERSION_ID}"; \
                    videolan_url="https://download.videolan.org/SuSE/${VERSION_ID}"; \
                    ;; \
                opensuse-tumbleweed) \
                    packman_url="https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Tumbleweed"; \
                    videolan_url="https://download.videolan.org/SuSE/Tumbleweed"; \
                    ;; \
                *) \
                    echo "ERROR: unsupported openSUSE ID: ${ID}" >&2; \
                    exit 1; \
                    ;; \
            esac; \
            rpm --import "${packman_url}/repodata/repomd.xml.key"; \
            zypper -n ar -f -c -p 90 "${packman_url}" packman; \
            rpm --import "${videolan_url}/repodata/repomd.xml.key"; \
            zypper -n ar -f -c -p 95 "${videolan_url}" videolan; \
            ;; \
    esac

# Refresh metadata before the validator queries the repositories, so stale
# cache cannot hide renamed or missing packages on dynamic base tags.
RUN zypper --non-interactive refresh

# Leap 16.1 images can ship patterns-base-fips, which conditionally pulls in
# crypto-policies-scripts and forces a downgrade of the vendor-provided
# crypto-policies package when openssh-clients is installed. Locking the
# pattern is not enough; it must be removed before validating packages.
RUN if rpm -q patterns-base-fips >/dev/null 2>&1; then \
        zypper --non-interactive remove -y patterns-base-fips; \
    fi

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Install the requested packages, update the system, remove orphaned
# packages, and clean Zypper metadata in the same layer. Nonfree builds
# add the Packman codec/media stack and VideoLAN's DVD CSS library.
RUN set -eu; \
    case "${TAG:-}" in \
        *-nonfree) extra_packages="gstreamer-plugins-bad-codecs gstreamer-plugins-ugly-codecs libx264-165 libfdk-aac2 libaacs0 libdvdcss2 vlc vlc-codecs" ;; \
        *) extra_packages="" ;; \
    esac; \
    zypper -n install --auto-agree-with-licenses -y \
    --allow-downgrade --allow-vendor-change \
    $(sh /tmp/pkg-validator.sh --pkgmgr zypper -- \
    bc \
    zsh \
    man \
    mtr \
    pam \
    zip \
    fish \
    bash \
    curl \
    zypper \
    less \
    lsof \
    pigz \
    sudo \
    time \
    tree \
    wget \
    Mesa \
    bzip2 \
    gnupg \
    glibc \
    rsync \
    unzip \
    words \
    xauth \
    procps \
    shadow \
    ffmpeg \
    libva2 \
    iputils \
    ncurses \
    systemd \
    python3 \
    libvpx2 \
    wayland \
    libdrm2 \
    Mesa-dri \
    hostname \
    keyutils \
    nss-mdns \
    pinentry \
    timezone \
    pipewire \
    libopus0 \
    xwayland \
    xdg-utils \
    diffutils \
    findutils \
    man-pages \
    pam-extra \
    gstreamer \
    libFLAC12 \
    libvulkan1 \
    glibc-i18n \
    util-linux \
    mesa-libva \
    wireplumber \
    libx264-164 \
    libx265-215 \
    glibc-locale \
    xdg-user-dirs \
    pipewire-jack \
    pipewire-alsa \
    glibc-i18ndata \
    libwayland-egl1 \
    bash-completion \
    libvulkan_intel \
    openssh-clients \
    libvulkan_radeon \
    glibc-locale-base \
    xdg-desktop-portal \
    libwayland-server0 \
    libwayland-client0 \
    system-group-wheel \
    util-linux-systemd \
    intel-media-driver \
    libva-intel-driver \
    glibc-locale-source \
    pipewire-pulseaudio \
    gstreamer-plugins-bad \
    gstreamer-plugins-base \
    gstreamer-plugins-good \
    gstreamer-plugins-ugly \
    gstreamer-plugins-libav \
    ${extra_packages}) || { rc=${?}; [ "${rc}" -ge 100 ] && [ "${rc}" -lt 200 ] && [ "${rc}" -ne 105 ]; } && \
    zypper --non-interactive update -y --auto-agree-with-licenses && \
    orphans="$(zypper --non-interactive packages --orphaned -i | awk -F'|' '/^i/{print $3}')" && \
    if [ -n "$orphans" ]; then \
        zypper --non-interactive rm -y $orphans; \
    else \
        echo "No orphaned packages to remove."; \
    fi && \
    zypper --non-interactive clean && \
    zypper clean --all \
    && rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*

# Locale setup — glibc-locale (installed via pkg script) pre-builds locales on Leap;
# localedef requires glibc-i18ndata charmap files which are not available on Leap 15.x
RUN echo "LANG=en_US.UTF-8" > /etc/locale.conf

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