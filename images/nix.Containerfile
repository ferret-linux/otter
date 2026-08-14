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

# Declarative package set, split into two tiers to avoid same-batch
# filename collisions. util-linux and coreutils both ship bin/kill;
# nettools and iproute2 both ship legacy net command names (route,
# ifconfig, netstat, etc). Installing them in the same
# `nix profile add --file` batch makes nix error out rather than
# pick a winner, since priority only arbitrates between separate
# profile elements, not members of one batch.
#
# NOTE: this step must run and be written to disk *before* the
# base-image profile packages (coreutils-full, findutils, etc.) are
# removed below -- those provide mkdir/cat/basic shell utilities
# that this RUN step itself depends on.
#
# Both files are kept in the final image under /etc/otter so users
# can inspect and adjust the package set after the image is built.
RUN mkdir -p /etc/otter && \
    cat > /etc/otter/packages-core.nix <<'EOF'
let
  pkgs = (builtins.getFlake "nixpkgs").legacyPackages.${builtins.currentSystem};
in
with pkgs;

[
  util-linux
  iproute2
]
EOF

RUN cat > /etc/otter/packages.nix <<'EOF'
let
  pkgs = (builtins.getFlake "nixpkgs").legacyPackages.${builtins.currentSystem};
in
with pkgs;

[
  bc
  xz
  gnutar
  zsh
  git
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
  xauth
  ffmpeg
  libcap
  shadow
  man-db
  tzdata
  iputils
  ncurses
  python3
  tcpdump
  pipewire
  keyutils
  man-pages
  coreutils
  xdg-utils
  diffutils
  findutils
  nettools
  vpl-gpu-rt
  xdg-user-dirs
  pipewire.jack
  vulkan-loader
  bash-completion
  pinentry-curses
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

# Remove the base image's pre-existing profile elements that overlap
# with the declarative set above (same package/version fighting over
# the same profile priority -- e.g. two builds of gnutar's bin/tar),
# then install the declarative set in the same RUN so there's no
# window where coreutils/findutils are missing but a later step still
# needs them. Deliberately NOT removed: nix, nss-cacert, iana-etc,
# gnugrep, gzip -- not redeclared below, and nix/nss-cacert are
# load-bearing for the nix CLI itself.
#
# util-linux and iproute2 install first at the higher-precedence
# priority (lower number) so they win filename collisions (kill,
# mount, route, etc.) against the rest of the set, which installs
# second at a lower-precedence priority.
RUN nix profile remove \
    --profile /nix/var/nix/profiles/default \
    bash-interactive coreutils-full curl findutils gnutar \
    less man-db openssh wget which git-minimal && \
    nix profile add \
    --profile /nix/var/nix/profiles/default \
    --priority 0 \
    --file /etc/otter/packages-core.nix && \
    nix profile add \
    --profile /nix/var/nix/profiles/default \
    --priority 5 \
    --file /etc/otter/packages.nix && \
    nix-index && \
    nix store gc && \
    nix-store --gc && \
    nix store verify && \
    nix-store --verify && \
    nix store optimise && \
    nix-store --optimise && \
    nix-collect-garbage -d && \
    nix profile wipe-history && \
    nix-env --delete-generations old && \
    nh clean all --keep 1 --keep-since 0s --optimise && \
    rm -rf /var/log/* /var/tmp/*

# Timezone default
# NOTE: no /usr/share/zoneinfo here -- nothing lands at FHS paths in
# this image. tzdata's zoneinfo is symlinked into the Nix profile
# tree instead, at /nix/var/nix/profiles/default/share/zoneinfo.
RUN ln -sf /nix/var/nix/profiles/default/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone