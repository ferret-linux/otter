# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.gentoo

# Configure portage to use binary packages by default
RUN echo 'FEATURES="getbinpkg"' >> /etc/portage/make.conf && \
    echo 'ACCEPT_LICENSE="*"' >> /etc/portage/make.conf

# Sync portage tree and fetch binary package keys
RUN emerge-webrsync && getuto

# Upgrade Python first in its own layer
RUN emerge --ask=n --quiet-build --getbinpkg -uDN dev-lang/python

# Upgrade portage in a separate layer so it starts with the new Python already in place
RUN emerge --ask=n --quiet-build --getbinpkg -uDN sys-apps/portage

# Upgrade all packages
RUN emerge --ask=n --autounmask-continue --quiet-build --getbinpkg -uDN @world

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going $(sh /tmp/pkg-validator.sh --pkgmgr emerge -- \
    app-crypt/gnupg \
    app-crypt/pinentry \
    app-shells/bash-completion \
    app-text/tree \
    dev-lang/python \
    app-arch/unrar \
    net-misc/curl \
    net-misc/wget \
    app-arch/pigz \
    media-video/pipewire \
    media-video/wireplumber \
    media-video/ffmpeg \
    media-libs/gstreamer \
    media-libs/gst-plugins-base \
    media-libs/gst-plugins-good \
    media-libs/gst-plugins-bad \
    media-libs/gst-plugins-ugly \
    media-libs/gst-libav \
    media-libs/x264 \
    media-libs/libvpx \
    media-libs/x265 \
    media-libs/opus \
    media-libs/flac \
    media-sound/lame \
    media-libs/mesa \
    sys-apps/diffutils \
    sys-apps/findutils \
    sys-apps/less \
    sys-apps/shadow \
    sys-apps/openrc \
    sys-apps/util-linux \
    sys-devel/bc \
    sys-libs/ncurses \
    sys-process/lsof \
    sys-process/procps \
    app-admin/sudo \
    x11-apps/xauth)

# Locale setup
RUN sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && eselect locale set en_US.utf8

# Timezone default
RUN echo "UTC" > /etc/timezone \
    && emerge --config sys-libs/timezone-data

# Cleanup
RUN rm -rf \
    /var/cache/distfiles/* \
    /var/cache/binpkgs/* \
    /var/log/* \
    /var/tmp/*