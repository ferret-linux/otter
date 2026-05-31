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

# Re-enable locale and docs
RUN rm -f /etc/dpkg/dpkg.cfg.d/excludes

# Upgrade all packages
RUN apt-get update && apt-get upgrade -y

# Install packages
RUN apt-get install -y --no-install-recommends \
    apt-utils bash-completion bc bzip2 curl dialog diffutils findutils \
    gnupg gnupg2 gpgsm hostname iproute2 iputils-ping keyutils less \
    libcap2-bin libkrb5-3 libegl-mesa0 libegl1 libgl1 \
    libgl1-mesa-glx libglx-mesa0 libnss-mdns libnss-myhostname \
    curl wget systemd libvulkan1 locales lsof man-db manpages \
    mesa-vulkan-drivers mtr ncurses-base openssh-client passwd pigz \
    pinentry-curses procps python3 rsync sudo tcpdump time traceroute \
    tree tzdata unzip util-linux wget xauth xz-utils zip systemd

# Locale setup
RUN sed -i "s|# en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen \
    && locale-gen \
    && update-locale LANG=en_US.UTF-8

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN apt-get clean \
    && rm -rf \
    /var/lib/apt/lists/* \
    /var/log/* \
    /var/tmp/*