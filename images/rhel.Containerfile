# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

# Pre-create otter dirs
RUN mkdir -p \
    /etc/profile.d \
    /etc/sudoers.d \
    /usr/lib/otter

# Re-enable docs
RUN sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf 2>/dev/null || \
    sed -i '/tsflags=nodocs/d' /etc/yum.conf 2>/dev/null || true

# Upgrade all packages
RUN microdnf upgrade -y

# Install packages
RUN microdnf install -y \
    bash-completion bc bzip2 cracklib-dicts curl diffutils findutils \
    glibc-all-langpacks glibc-common glibc-locale-source gnupg2 \
    gnupg2-smime hostname iproute iputils keyutils krb5-libs less lsof \
    man-db mesa-dri-drivers mesa-vulkan-drivers mtr ncurses \
    openssh-clients pam pigz pinentry procps-ng python3 \
    rsync shadow-utils sudo time tzdata unzip \
    util-linux vte-profile vulkan-tools wget which \
    xorg-x11-xauth xz zip systemd

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN microdnf clean all \
    && rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*