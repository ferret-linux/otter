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

# Run package install script
COPY images/scripts/packages/pkg-opensuse.sh /tmp/pkg-opensuse.sh
RUN sh /tmp/pkg-opensuse.sh

# Locale setup — glibc-locale (installed via pkg script) pre-builds locales on Leap;
# localedef requires glibc-i18ndata charmap files which are not available on Leap 15.x
RUN echo "LANG=en_US.UTF-8" > /etc/locale.conf

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN zypper clean --all \
    && rm -rf \
    /var/cache/zypp/* \
    /var/log/* \
    /var/tmp/*