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
RUN touch /usr/lib/otter/container.amazonlinux

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || true

# Run package install script
COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Refresh metadata, upgrade the system, install the requested
# packages, and clean DNF metadata/cache in the same layer.
# NOTE: unlike rhel-family, this does not install EPEL - Amazon
# Linux 2023 does not define the %rhel rpm macro and is not an
# officially EPEL-supported distro.
RUN dnf clean expire-cache && \
    dnf makecache --refresh && \
    dnf upgrade -y && \
    dnf install -y --skip-broken $(sh /tmp/pkg-validator.sh --pkgmgr dnf -- \
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
        hostname \
        keyutils \
        pinentry \
        nss-mdns \
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
        cracklib-dicts \
        xorg-x11-xauth \
        bash-completion \
        openssh-clients \
        dnf-plugins-core \
        mesa-dri-drivers \
        util-linux-script \
        glibc-all-langpacks \
        glibc-locale-source \
        mesa-vulkan-drivers) && \
    dnf upgrade -y && \
    dnf autoremove -y && \
    dnf clean all && \
    rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh