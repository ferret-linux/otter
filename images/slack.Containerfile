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
RUN touch /usr/lib/otter/container.slackware

# NOTE: no mirror-selection step needed here - aclemons/slackware
# ships with slackpkg already pointed at the correct mirror for
# whichever release this tag actually is ("slackpkg needs to work
# out of the box" is one of that image's stated goals).

# Run non-interactively (should we make it default?)
#RUN sed -i 's/^DIALOG=.*/DIALOG=off/' /etc/slackpkg/slackpkg.conf 2>/dev/null || \
#    echo "DIALOG=off" >> /etc/slackpkg/slackpkg.conf

COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# NOTE: slackpkg does not perform automatic dependency resolution -
# unlike apt/dnf/pacman, packages not explicitly listed here will
# not be pulled in as transitive dependencies. This list may need
# to be more exhaustive than other distros' equivalents.
#
# NOTE: default BSD-style init (/etc/rc.d, sysvinit package) is
# already present as part of Slackware's mandatory base "A" series -
# nothing installed or changed here. systemd is intentionally not
# in this list (not part of Slackware)
RUN printf 'yes\n' | DIALOG=off slackpkg update gpg && \
    DIALOG=off slackpkg update && \
    for pkg in $(sh /tmp/pkg-validator.sh --pkgmgr slackpkg -- \
        bc \
        xz \
        git \
        mtr \
        tar \
        zsh \
        bash \
        curl \
        fish \
        gzip \
        lame \
        less \
        lsof \
        mesa \
        sudo \
        time \
        tree \
        wget \
        bzip2 \
        gnupg \
        rsync \
        unzip \
        which \
        dialog \
        ffmpeg \
        libdrm \
        libvpx \
        man-db \
        shadow \
        tzdata \
        infozip \
        ncurses \
        openssh \
        python3 \
        tcpdump \
        wayland \
        hostname \
        iproute2 \
        keyutils \
        libglvnd \
        libinput \
        nss-mdns \
        pinentry \
        pipewire \
        diffutils \
        findutils \
        gstreamer \
        procps-ng \
        util-linux \
        wireplumber \
        bash-completion \
        gst-plugins-base \
        gst-plugins-good \
        gst-plugins-libav \
        wayland-protocols \
        xdg-desktop-portal \
        gst-plugins-bad-free \
        xorg-server-xwayland); do \
        DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install "${pkg}"; \
    done

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh