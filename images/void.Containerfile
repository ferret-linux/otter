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

# Configure xbps to not overwrite host-managed files
RUN mkdir -p /etc/xbps.d \
    && echo -e "noextract=/etc/passwd\nnoextract=/etc/hosts\nnoextract=/etc/host.conf\nnoextract=/etc/hostname\nnoextract=/etc/localtime\nnoextract=/etc/machine-id\nnoextract=/etc/resolv.conf" \
    > /etc/xbps.d/otter-ignore.conf

# Update xbps itself first
RUN xbps-install -Syu xbps

# Upgrade all packages
RUN xbps-install -Syu
# Run package install script
COPY images/scripts/packages/pkg-void.sh /tmp/pkg-void.sh
RUN sh /tmp/pkg-void.sh

# Locale setup (glibc only, musl does not use libc-locales)
RUN if [ -f /etc/default/libc-locales ]; then \
    sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/default/libc-locales \
    && xbps-reconfigure --force glibc-locales; \
    fi

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/xbps/* \
    /var/log/* \
    /var/tmp/*