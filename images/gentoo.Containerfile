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
        app-text/tree \
        net-misc/curl \
        net-misc/wget \
        app-arch/pigz \
        sys-apps/less \
        app-arch/unrar \
        app-shells/zsh \
        app-admin/sudo \
        x11-apps/xauth \
        app-shells/fish \
        app-crypt/gnupg \
        dev-lang/python \
        media-libs/x264 \
        media-libs/x265 \
        media-libs/opus \
        media-libs/flac \
        media-libs/mesa \
        app-shells/bash \
        sys-apps/shadow \
        sys-apps/openrc \
        media-sound/lame \
        sys-libs/ncurses \
        sys-process/lsof \
        media-libs/libvpx \
        app-crypt/pinentry \
        media-video/ffmpeg \
        sys-apps/diffutils \
        sys-apps/findutils \
        sys-process/procps \
        sys-apps/util-linux \
        media-video/pipewire \
        media-libs/gstreamer \
        media-libs/gst-libav \
        app-portage/gentoolkit \
        x11-libs/libXi \
        media-libs/tiff \
        x11-libs/libX11 \
        x11-libs/libxcb \
        media-libs/libva \
        x11-libs/libXext \
        media-libs/libaom \
        media-libs/libpng \
        x11-base/xwayland \
        media-libs/libexif \
        media-libs/libheif \
        media-libs/libwebp \
        x11-libs/libXfixes \
        x11-libs/libXrandr \
        media-libs/libdav1d \
        media-libs/libde265 \
        media-libs/libvdpau \
        media-libs/openh264 \
        x11-libs/libXcursor \
        x11-libs/libXdamage \
        x11-libs/libXinerama \
        x11-libs/libxkbcommon \
        x11-libs/libXcomposite \
        media-libs/libjpeg-turbo \
        media-libs/vulkan-layers \
        media-libs/vulkan-loader \
        gui-apps/xdg-desktop-portal \
        media-video/wireplumber \
        app-shells/bash-completion \
        media-libs/gst-plugins-bad \
        media-libs/gst-plugins-base \
        media-libs/gst-plugins-good \
        media-libs/gst-plugins-ugly) && \
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
