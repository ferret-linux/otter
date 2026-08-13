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

# Sync portage tree and fetch binary package keys
RUN emerge-webrsync && getuto

# Upgrade Python first in its own layer
RUN emerge --ask=n --quiet-build --getbinpkg -uDN dev-lang/python dev-lang/rust-bin

# Upgrade portage in a separate layer so it starts with the new Python already in place
RUN emerge --ask=n --quiet-build --getbinpkg -uDN sys-apps/portage

# Upgrade all packages
RUN emerge --ask=n --autounmask-continue --quiet-build --getbinpkg -uDN @world

# Run package install script (fish already installed above, removed from this list)
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh
RUN emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going $(sh /tmp/pkg-validator.sh --pkgmgr emerge -- \
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
    media-video/wireplumber \
    app-shells/bash-completion \
    media-libs/gst-plugins-bad \
    media-libs/gst-plugins-base \
    media-libs/gst-plugins-good \
    media-libs/gst-plugins-ugly)

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

# Portage cleanup
RUN emerge --sync && \
    emerge -uDN @world && \
    emerge --depclean && \
    eclean distfiles && \
    eclean packages

# Cleanup
RUN rm -rf \
    /var/cache/distfiles/* \
    /var/cache/binpkgs/* \
    /var/log/* \
    /var/tmp/*