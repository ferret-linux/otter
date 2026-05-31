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
RUN dnf upgrade -y

# Run package install script
COPY images/scripts/pkg_rhel_family.sh /tmp/pkg_rhel_family.sh
RUN sh /tmp/pkg_rhel_family.sh

# Locale setup
RUN localedef -i en_US -f UTF-8 en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN dnf clean all \
    && rm -rf \
    /var/cache/dnf/* \
    /var/log/* \
    /var/tmp/*