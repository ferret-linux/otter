# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# This image ships no /etc/os-release at all (confirmed via
# `cat /etc/os-release` -> "No such file or directory"). Write our
# own so get_distro_id() resolves to a known value instead of
# otter-init hard-exiting with 127.
RUN printf 'ID=nix\nNAME="NixOS"\nPRETTY_NAME="NixOS OCI"\n' > /etc/os-release

# Flakes / the `nix` CLI are not enabled by default here (confirmed:
# /etc/nix/nix.conf ships with no experimental-features line).
# Append rather than overwrite, to preserve the existing
# build-users-group / sandbox / trusted-public-keys lines already
# in the image.
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.nix

# Pin the "nixpkgs" flake registry entry explicitly to the unstable
# channel branch, so every `nixpkgs#pkg` reference below (and
# anything a user runs later) resolves deterministically instead of
# whatever the global flake registry's default happens to be.
RUN nix registry pin nixpkgs github:NixOS/nixpkgs/nixpkgs-unstable

# Install the package set into the default profile.
# Not run through pkg-validator.sh (agreed) -- Nix resolves/builds
# each attribute at install time, so a bad name fails the build
# directly instead of needing pre-validation.
RUN nix profile install \
    nixpkgs#bc \
    nixpkgs#xz \
    nixpkgs#gnutar \
    nixpkgs#zsh \
    nixpkgs#gnupg \
    nixpkgs#zip \
    nixpkgs#bashInteractive \
    nixpkgs#fish \
    nixpkgs#curl \
    nixpkgs#less \
    nixpkgs#lsof \
    nixpkgs#pigz \
    nixpkgs#sudo \
    nixpkgs#tree \
    nixpkgs#vte \
    nixpkgs#wget \
    nixpkgs#which \
    nixpkgs#bzip2 \
    nixpkgs#rsync \
    nixpkgs#unzip \
    nixpkgs#xorg.xauth \
    nixpkgs#ffmpeg \
    nixpkgs#libcap \
    nixpkgs#shadow \
    nixpkgs#mandoc \
    nixpkgs#tzdata \
    nixpkgs#iputils \
    nixpkgs#ncurses \
    nixpkgs#python3 \
    nixpkgs#tcpdump \
    nixpkgs#pipewire \
    nixpkgs#iproute2 \
    nixpkgs#keyutils \
    nixpkgs#pinentry \
    nixpkgs#man-pages \
    nixpkgs#coreutils \
    nixpkgs#xdg-utils \
    nixpkgs#diffutils \
    nixpkgs#findutils \
    nixpkgs#nettools \
    nixpkgs#vpl-gpu-rt \
    nixpkgs#util-linux \
    nixpkgs#xdg-user-dirs \
    nixpkgs#pipewire.jack \
    nixpkgs#vulkan-loader \
    nixpkgs#bash-completion \
    nixpkgs#gst_all_1.gstreamer \
    nixpkgs#gst_all_1.gst-plugins-bad \
    nixpkgs#gst_all_1.gst-plugins-base \
    nixpkgs#gst_all_1.gst-plugins-ugly \
    nixpkgs#gst_all_1.gst-plugins-good \
    nixpkgs#xdg-desktop-portal \
    nixpkgs#openssh

# Install nh, comma, and nix-index, then build the nix-index
# database at image-build time (agreed: baked in, not built lazily
# on first run).
RUN nix profile install nixpkgs#nh nixpkgs#comma nixpkgs#nix-index \
    && nix-index

# Timezone default
# NOTE: no /usr/share/zoneinfo here -- nothing lands at FHS paths in
# this image. tzdata's zoneinfo is symlinked into the Nix profile
# tree instead, at /root/.nix-profile/share/zoneinfo (confirmed via
# the earlier PATH/profile inspection).
RUN ln -sf /root/.nix-profile/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Nix store cleanup: drop build-time garbage not reachable from the
# current profile generation, then hardlink duplicate store paths
# to shrink the image.
RUN nix store gc && \
    nix-store --gc &&\ 
    nix store verify && \ 
    nix-store --verify && \
    nix store optimise && \
    nix-store --optimise && \
    nix-collect-garbage -d && \
    nix profile wipe-history && \
    nix-env --delete-generations old && \
    nh clean all --non-interactive --keep 1 --keep-since 0s

# Cleanup
RUN rm -rf \
    /var/log/* \
    /var/tmp/*