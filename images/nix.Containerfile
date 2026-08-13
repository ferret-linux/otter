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

# Declarative package set.
#
# This file is intentionally kept in the final image at
# /etc/otter/packages.nix so users can inspect and configure the
# package set after the image is built.
RUN mkdir -p /etc/otter && \
    cat > /etc/otter/packages.nix <<'EOF'
let
  pkgs = builtins.getFlake "nixpkgs";
in
with pkgs;

[
  bc
  xz
  gnutar
  zsh
  gnupg
  zip
  bashInteractive
  fish
  curl
  less
  lsof
  pigz
  sudo
  tree
  vte
  wget
  which
  bzip2
  rsync
  unzip
  xorg.xauth
  ffmpeg
  libcap
  shadow
  mandoc
  tzdata
  iputils
  ncurses
  python3
  tcpdump
  pipewire
  iproute2
  keyutils
  pinentry
  man-pages
  coreutils
  xdg-utils
  diffutils
  findutils
  nettools
  vpl-gpu-rt
  util-linux
  xdg-user-dirs
  pipewire.jack
  vulkan-loader
  bash-completion
  gst_all_1.gstreamer
  gst_all_1.gst-plugins-bad
  gst_all_1.gst-plugins-base
  gst_all_1.gst-plugins-ugly
  gst_all_1.gst-plugins-good
  xdg-desktop-portal
  openssh
  nh
  comma
  nix-index
]
EOF

# Install the complete package set from the declarative file into
# the system/default Nix profile.
RUN nix profile install \
    --profile /nix/var/nix/profiles/default \
    --file /etc/otter/packages.nix

# Build the nix-index database at image-build time
# (agreed: baked in, not built lazily on first run).
RUN nix-index

# Timezone default
# NOTE: no /usr/share/zoneinfo here -- nothing lands at FHS paths in
# this image. tzdata's zoneinfo is symlinked into the Nix profile
# tree instead, at /nix/var/nix/profiles/default/share/zoneinfo.
RUN ln -sf /nix/var/nix/profiles/default/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Nix store cleanup: drop build-time garbage not reachable from the
# current profile generation, then hardlink duplicate store paths
# to shrink the image.
RUN nix store gc && \
    nix-store --gc && \
    nix store verify && \
    nix-store --verify && \
    nix store optimise && \
    nix-store --optimise && \
    nix-collect-garbage -d && \
    nix profile wipe-history && \
    nix-env --delete-generations old && \
    nh clean all --keep 1 --keep-since 0s --optimise

# Cleanup
RUN rm -rf \
    /var/log/* \
    /var/tmp/*