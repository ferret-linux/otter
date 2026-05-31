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

# Configure portage to use binary packages by default
RUN echo 'FEATURES="getbinpkg"' >> /etc/portage/make.conf

# Sync portage tree and fetch binary package keys
RUN emerge-webrsync && getuto

# Upgrade all packages
RUN emerge --ask=n --autounmask-continue --quiet-build --getbinpkg -uDN @world

# Run package install script
COPY images/scripts/pkg_gentoo.sh /tmp/pkg_gentoo.sh
RUN sh /tmp/pkg_gentoo.sh

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