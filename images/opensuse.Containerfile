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

# Configure zypper to install recommended packages and docs
RUN mkdir -p /etc/zypp/zypp.conf.d \
    && echo -e "[main]\nsolver.onlyRequires = false\nrpm.install.excludedocs = no" \
    > /etc/zypp/zypp.conf.d/99-otter.conf

# Lock parallel-printer-support as it can't be installed in rootless containers
RUN zypper al parallel-printer-support

# Upgrade all packages
RUN zypper dup -y

# Install packages
RUN zypper -n install -y \
    bash-completion bc bzip2 curl diffutils findutils glibc-locale \
    glibc-locale-base hostname iputils keyutils less lsof man \
    man-pages Mesa-dri libvulkan1 libvulkan_intel libvulkan_radeon \
    libvte-2_91-0 mtr nss-mdns openssh-clients pam pam-extra \
    pigz pinentry procps rsync shadow sudo system-group-wheel \
    systemd time timezone tree unzip util-linux util-linux-systemd \
    wget words xauth zip

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN zypper clean --all \
    && rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*