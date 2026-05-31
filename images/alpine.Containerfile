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

# Upgrade all packages
RUN apk update && apk upgrade

# Install packages
RUN apk add --no-cache \
    bash bash-completion bc bzip2 coreutils curl diffutils docs \
    findutils gcompat gnupg gpg iproute2 iputils keyutils less \
    libcap libc-utils lsof man-pages mandoc mesa-dri-gallium \
    mesa-vulkan-swrast musl-utils ncurses ncurses-terminfo net-tools \
    openssh-client-default pigz pinentry procps python3 rsync shadow \
    sudo tar tcpdump tree tzdata umount unzip util-linux util-linux-login \
    util-linux-misc vulkan-loader vte3 wget which xauth xz zip openrc

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Cleanup
RUN rm -rf \
    /var/cache/apk/* \
    /var/log/* \
    /var/tmp/*