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
RUN touch /usr/lib/otter/container.gentoo

# Configure portage to use binary packages by default
RUN echo 'FEATURES="getbinpkg"' >> /etc/portage/make.conf && \
    echo 'ACCEPT_LICENSE="*"' >> /etc/portage/make.conf

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Sync portage tree, fetch binary package keys, upgrade Python and
# Portage, upgrade all packages, install requested packages, and
# perform Portage cleanup in the same layer.
RUN emerge-webrsync && \
    getuto && \
    emerge --ask=n --quiet-build --getbinpkg -uDN dev-lang/python dev-lang/rust-bin && \
    emerge --ask=n --quiet-build --getbinpkg -uDN sys-apps/portage && \
    emerge --ask=n --autounmask-continue --quiet-build --getbinpkg -uDN @world && \
    emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going $(sh /tmp/pkg-validator.sh --pkgmgr emerge -- \
        sys-devel/bc \
        app-arch/pigz \
        app-text/tree \
        net-misc/curl \
        net-misc/wget \
        sys-apps/less \
        app-admin/sudo \
        app-arch/unrar \
        app-shells/zsh \
        x11-apps/xauth \
        x11-libs/libXi \
        app-crypt/gnupg \
        app-shells/bash \
        app-shells/fish \
        dev-lang/python \
        media-libs/flac \
        media-libs/mesa \
        media-libs/opus \
        media-libs/tiff \
        media-libs/x264 \
        media-libs/x265 \
        sys-apps/openrc \
        sys-apps/shadow \
        x11-libs/libX11 \
        x11-libs/libxcb \
        x11-libs/libdrm \
        media-libs/dav1d \
        media-libs/libva \
        media-sound/lame \
        sys-libs/ncurses \
        sys-process/lsof \
        x11-libs/libXext \
        x11-libs/libvdpau \
        media-libs/libaom \
        media-libs/libpng \
        media-libs/libvpx \
        x11-base/xwayland \
        x11-misc/xdg-utils \
        app-crypt/pinentry \
        media-libs/libexif \
        media-libs/libheif \
        media-libs/libwebp \
        media-video/ffmpeg \
        sys-apps/diffutils \
        sys-apps/findutils \
        sys-process/procps \
        x11-libs/libXfixes \
        x11-libs/libXrandr \
        media-libs/libde265 \
        media-libs/openh264 \
        sys-apps/util-linux \
        x11-libs/libXcursor \
        x11-libs/libXdamage \
        media-libs/gstreamer \
        media-video/pipewire \
        x11-libs/libXinerama \
        x11-libs/libxkbcommon \
        x11-misc/xdg-user-dirs \
        app-portage/gentoolkit \
        x11-libs/libXcomposite \
        media-video/wireplumber \
        media-libs/libjpeg-turbo \
        media-libs/vulkan-layers \
        media-libs/vulkan-loader \
        app-shells/bash-completion \
        media-libs/gst-plugins-bad \
        sys-apps/xdg-desktop-portal \
        media-libs/gst-plugins-base \
        media-libs/gst-plugins-good \
        media-libs/gst-plugins-ugly \
        media-plugins/gst-plugins-libav) && \
    emerge --sync && \
    emerge --with-bdeps=y -uDN @world && \
    emerge --depclean && \
    eclean distfiles && \
    eclean packages && \
    rm -rf \
    /var/cache/distfiles/* \
    /var/cache/binpkgs/* \
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
RUN sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && eselect locale set en_US.utf8

# Timezone default
RUN echo "UTC" > /etc/timezone \
    && emerge --config sys-libs/timezone-data

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