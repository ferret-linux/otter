# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE

# Resolve stage: determines which nixpkgs flake ref to pin further
# down, isolated in its own stage the same way dinit is resolved and
# built in its own stage in wolfi.Containerfile. A stage's only way
# to hand a computed value (as opposed to a build tool's output) to
# another stage is via the filesystem, so the resolved ref is
# written to a plain file here and COPY --from='d into the main
# stage below.
FROM ${IMAGE} AS channel-resolver
ARG TAG

# TAG selects which channel: "unstable" pins the rolling
# nixpkgs-unstable branch. "stable" resolves the current stable
# NixOS release cycle at build time (e.g. "26.05") via the
# endoflife.date API, so this stays correct across future NixOS
# releases without needing a manual bump here. curl is already
# present in this base image's default profile, so no extra install
# is needed. If the lookup fails for any reason, fall back to a
# hardcoded last-known stable version rather than failing the build.
RUN if [ "$TAG" = "stable" ]; then \
        STABLE_CYCLE="$(curl -fsSL https://endoflife.date/api/nixos.json | grep -o '"cycle": *"[^"]*"' | head -1 | cut -d'"' -f4)"; \
        if [ -z "${STABLE_CYCLE}" ]; then STABLE_CYCLE="26.05"; fi; \
        echo "Resolved stable channel: ${STABLE_CYCLE}"; \
        echo "${STABLE_CYCLE}" > /channel-ref; \
    else \
        echo "Resolved unstable channel: nixpkgs-unstable"; \
        echo "nixpkgs-unstable" > /channel-ref; \
    fi

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
# Rebuilt from scratch rather than appended, pulling forward only
# trusted-public-keys from the base image's original nix.conf (that
# key is base-image/build-specific and shouldn't be hardcoded here);
# build-users-group and sandbox are set explicitly below instead of
# inherited, since this file already needs to declare them anyway.
# Container builds can't rely on Nix's default build sandbox --
# user namespaces / bubblewrap are frequently unavailable when Nix
# is already running inside another container -- and interactive
# users doing local `nix build`/`nix develop` work later expect GC
# to leave their in-progress build's dependencies alone rather than
# sweeping them up the moment a derivation is realized.
#
# filter-syscalls installs a seccomp-BPF filter before builds run,
# independent of `sandbox` above -- it blocks setuid/setgid and
# capability/ACL manipulation in build outputs as a hardening
# measure. That BPF program cannot be loaded under qemu-user
# emulation (cross-arch builds, e.g. building arm64 on an amd64
# host or vice versa, which otter explicitly supports) -- Nix fails
# with "unable to load seccomp BPF program: Invalid argument"
# regardless of hardware. Disabled here because otter needs
# emulated cross-arch builds to work at all; this is not
# recoverable by upgrading Nix/qemu/kernel, only by building
# natively per-arch instead, which isn't otter's model.
# auto-optimise-store hardlinks each new store path against the
# existing store incrementally, at add time, instead of leaving it
# for a single retroactive `nix store optimise` pass over the whole
# store later. The retroactive pass does thousands of renames in one
# burst and is what was triggering "Stale file handle" errors against
# GitHub-hosted runners' overlayfs (see containers/podman#23808,
# containers/podman#5816) -- incremental per-path hardlinking avoids
# that burst pattern while still getting the same disk savings.
#
# Both settings blocks below are written in one RUN via a
# read-modify-write against a fresh /tmp file rather than appending
# directly to /etc/nix/nix.conf: on this base image, that path is a
# symlink into the Nix store, and shell `>>` redirection follows
# symlinks, so an in-place append would silently write into an
# immutable store path (root bypasses the store's read-only file
# perms) and corrupt it -- `rm -f` on a symlink only removes the
# symlink itself, never the store target, which is what makes this
# safe.
RUN grep '^trusted-public-keys' /etc/nix/nix.conf > /tmp/nix.conf.new && \
    printf '%s\n' \
    'sandbox = false' \
    'keep-outputs = true' \
    'filter-syscalls = false' \
    'keep-derivations = true' \
    'auto-optimise-store = true' \
    'build-users-group = nixbld' \
    'experimental-features = nix-command flakes' \
    >> /tmp/nix.conf.new && \
    rm -f /etc/nix/nix.conf && \
    mv /tmp/nix.conf.new /etc/nix/nix.conf

# The base image's closure (glibc, bash-interactive, coreutils, nix
# itself, systemd-minimal-libs, etc.) is registered directly into
# /nix/var/nix/db by buildLayeredImage at base-image build time,
# without ever going through the substituter/binary-cache protocol
# -- so no narinfo signature is attached locally, and `nix-store
# --verify` / `nix store verify --all` report every one of those
# paths as "untrusted" even though the content itself is fine.
# `nix copy` alone does not fix this: it skips transferring
# signatures for paths that already exist in the destination store
# (see NixOS/nix#4221). `nix store copy-sigs` is the command built
# for exactly this case -- it fetches and attaches signatures for
# paths already present locally, without re-copying their content.
# Must run before the first `nix store verify --all` call.
RUN nix store copy-sigs --substituter https://cache.nixos.org --recursive $(nix-store -qR /nix/var/nix/profiles/default)

# A handful of paths in the base image's closure (e.g. base-system,
# channel-nixos, the nixpkgs source checkout, libbsd, libmd, shadow,
# tcb, glibc's -bin output) were composed locally by the base
# image's own build tooling rather than fetched from cache.nixos.org,
# so copy-sigs above has no upstream signature to fetch for them --
# they remain "untrusted" to `nix store verify --all` even though
# their contents are intact. Sign them with a throwaway local key
# instead: the key is generated fresh on every build (no need to
# persist it across rebuilds), its public half is appended to
# trusted-public-keys so verify accepts its signatures, the closure
# is signed, and the secret key is deleted immediately after so it
# never ends up in the final image.
RUN nix key generate-secret --key-name otter-local-1 > /tmp/otter-local-signing-key && \
    printf 'extra-trusted-public-keys = %s\n' "$(nix key convert-secret-to-public < /tmp/otter-local-signing-key)" >> /etc/nix/nix.conf && \
    nix store sign --key-file /tmp/otter-local-signing-key --all && \
    rm -f /tmp/otter-local-signing-key

# Verify store paths/packages
RUN nix-store --verify && \
    nix store verify --all

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.nix

# Pin the "nixpkgs" flake registry entry explicitly to the channel
# resolved above, so every `nixpkgs#pkg` reference below (and
# anything a user runs later) resolves deterministically instead of
# whatever the global flake registry's default happens to be.
COPY --from=channel-resolver /channel-ref /tmp/channel-ref
RUN nix registry pin nixpkgs github:NixOS/nixpkgs/$(cat /tmp/channel-ref) && \
    rm /tmp/channel-ref

# Declarative package set, split into five theme-focused tiers
# (roughly evenly sized) to avoid same-batch filename collisions.
# Packages that ship overlapping binaries can't be installed in the
# same `nix profile add --file` batch -- nix errors out rather than
# pick a winner, since priority only arbitrates between separate
# profile elements, not members of one batch. Known collisions
# driving the split, each pair kept in different files below:
#   - util-linux vs coreutils: both ship bin/kill
#   - util-linux vs shadow: both ship bin/chfn (and chsh, login)
#   - shadow vs man-pages: both ship share/man/man3/getspnam.3.gz
#   - tzdata vs man-pages: both ship share/man/man5/tzfile.5.gz
#   - nettools vs iproute2: both ship legacy net command names
#     (route, ifconfig, netstat, etc)
# Satisfying all five pairs across five theme-based files forces two
# packages out of their natural theme: coreutils-full lives in
# "network" (not "core") since core already holds util-linux, and
# tzdata lives in "multimedia" (not "utilities") since "utilities"
# already holds man-pages. Every other file's contents are chosen by
# theme.
#
# core: system/process management, installed at the highest
# precedence (lowest priority number) so it wins any binary
# collision against the other four tiers.
#
# shell: shells and developer tooling, installed second.
#
# multimedia: graphics/audio/desktop-integration packages, installed
# third.
#
# network: networking tools plus core CLI utilities (coreutils-full
# lives here, see above), installed fourth.
#
# utilities: archive tools, docs, locale data, and misc (man-pages
# lives here, see above), installed last at the lowest precedence,
# so it's always the side that gets silently shadowed on any binary
# it happens to share with an earlier tier.
#
# All five files are kept in the final image under /etc/otter so
# users can inspect and adjust the package set after the image is
# built.
#
# nix-settings.nix holds the nixpkgs `config` attrset (currently
# just allowUnfree) applied when packages-lib.nix instantiates pkgs.
# Kept as its own file, alongside the package tiers, so it's
# editable in the same place users already go to adjust packages.
#
# packages-lib.nix is a shared helper imported by all five tiers
# below: it centralizes the `pkgs` binding (now instantiated via
# `import` with nix-settings.nix's config, rather than reaching
# through the flake's pre-instantiated `legacyPackages`, since
# `legacyPackages` has a fixed config baked in and can't have
# allowUnfree applied after the fact) and an `onlySupported` filter
# (built on nixpkgs' own lib.meta.availableOn) that drops any
# package not available on the current build arch before it's
# handed to `nix profile add` -- e.g. vpl-gpu-rt, which nixpkgs
# marks x86_64-linux-only, would otherwise abort evaluation on
# aarch64-linux builds with "Refusing to evaluate package ... not
# available on the requested hostPlatform".
RUN mkdir -p /etc/otter && \
    cat > /etc/otter/nix-settings.nix <<'EOF'
{
  allowUnfree = true;
}
EOF

RUN cat > /etc/otter/packages-lib.nix <<'EOF'
let
  settings = import /etc/otter/nix-settings.nix;
  pkgs = import (builtins.getFlake "nixpkgs") {
    system = builtins.currentSystem;
    config = settings;
  };
  onlySupported = builtins.filter (pkgs.lib.meta.availableOn pkgs.stdenv.hostPlatform);
in
{
  inherit pkgs onlySupported;
}
EOF

RUN cat > /etc/otter/packages-core.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  bc
  lsof
  sudo
  tree
  gnused
  libcap
  python3
  ncurses
  systemd
  iproute2
  keyutils
  findutils
  util-linux
  bash-completion
]
EOF

RUN cat > /etc/otter/packages-shell.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  nh
  git
  zsh
  fish
  comma
  gnupg
  man-db
  shadow
  python3
  nix-index
  bashInteractive
  pinentry-curses
]
EOF

RUN cat > /etc/otter/packages-multimedia.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  mesa
  tzdata
  libglvnd
  pipewire
  vpl-gpu-rt
  wireplumber
  pipewire.jack
  vulkan-loader
  xdg-desktop-portal
  gst_all_1.gstreamer
  gst_all_1.gst-plugins-bad
  gst_all_1.gst-plugins-base
  gst_all_1.gst-plugins-good
  gst_all_1.gst-plugins-ugly
]
EOF

RUN cat > /etc/otter/packages-network.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  xz
  curl
  krb5
  wget
  bzip2
  rsync
  iputils
  openssh
  tcpdump
  nettools
  diffutils
  traceroute
  glibc.getent
  coreutils-full
]
EOF

RUN cat > /etc/otter/packages-utilities.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  vte
  zip
  less
  pigz
  unzip
  which
  xauth
  ffmpeg
  gnutar
  procps
  wayland
  xwayland
  man-pages
  xdg-utils
  glibcLocales
  xdg-user-dirs
]
EOF

# Remove the base image's pre-existing profile elements that overlap
# with the declarative set above (same package/version fighting over
# the same profile priority -- e.g. two builds of gnutar's bin/tar),
# then install the five-tier declarative set in the same RUN so
# there's no window where coreutils/findutils are missing but a
# later step still needs them. Deliberately NOT removed: nix,
# nss-cacert, iana-etc, gnugrep, gzip -- not redeclared below, and
# nix/nss-cacert are load-bearing for the nix CLI itself.
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
    --priority 1 \
    --file /etc/otter/packages-shell.nix && \
    nix profile add \
    --profile /nix/var/nix/profiles/default \
    --priority 2 \
    --file /etc/otter/packages-multimedia.nix && \
    nix profile add \
    --profile /nix/var/nix/profiles/default \
    --priority 3 \
    --file /etc/otter/packages-network.nix && \
    nix profile add \
    --profile /nix/var/nix/profiles/default \
    --priority 4 \
    --file /etc/otter/packages-utilities.nix && \
    nix store gc && \
    nix-store --gc && \
    nix-store --verify && \
    nix store optimise && \
    nix-store --optimise && \
    nix store verify --all && \
    nix-collect-garbage -d && \
    nix profile wipe-history && \
    nix-env --delete-generations old && \
    rm -rf /var/log/* /var/tmp/*

# This image has no NixOS module system to generate PAM service files
# or /etc/login.defs the way a real NixOS install would, so shadow's
# password tools (passwd, chpasswd) have no policy to authenticate
# against and fail outright. shadow's own PAM linkage is left enabled
# (not nulled out) so this matches every other image, which all get
# working PAM-backed passwd/chpasswd from their distro's package
# manager. pam_unix.so is referenced by bare name, same as the
# existing pam-su template (otter-init) -- this resolves correctly
# against this build's own Nix store path because libpam compiles its
# module search path in at build time (DEFAULT_MODULE_PATH), rather
# than reading it from the config file, so no store path needs to be
# hardcoded here.
#
# ENCRYPT_METHOD is pinned to SHA512 explicitly, rather than relying
# on whatever this shadow build's compiled-in default resolves to,
# since it's supported unconditionally by every libxcrypt build, with
# or without the optional yescrypt/bcrypt modules this build also
# enables.
RUN mkdir -p /etc/pam.d && \
    printf '%s\n' \
    'password    required    pam_unix.so' \
    > /etc/pam.d/passwd && \
    printf '%s\n' \
    'password    required    pam_unix.so' \
    > /etc/pam.d/chpasswd && \
    printf '%s\n' \
    'ENCRYPT_METHOD SHA512' \
    > /etc/login.defs

# otter-init looks for the systemd binary and unit files at the FHS
# paths /usr/lib/systemd/systemd and /usr/lib/systemd/system/*, and
# expects `systemctl` on $PATH (already satisfied -- the profile's
# bin dir is on $PATH via this base image's own ONBUILD ENV). Nix
# profile-installs systemd's own output tree under
# <profile>/lib/systemd instead of at those FHS paths, so symlink
# the whole directory across in one shot rather than each file
# individually.
RUN mkdir -p /usr/lib && \
    ln -sf /nix/var/nix/profiles/default/lib/systemd /usr/lib/systemd

# glibc on non-NixOS systems can't assume an FHS locale-archive path,
# so nixpkgs' glibc is patched to read from $LOCALE_ARCHIVE instead
# (documented nixpkgs behavior, not a workaround specific to this
# image). Without this, locale-dependent tools emit "cannot change
# locale" warnings or misbehave.
ENV LANG=en_US.UTF-8
ENV LOCALE_ARCHIVE=/nix/var/nix/profiles/default/lib/locale/locale-archive

# Lets classic-style commands (nix-env, nix-build, nix-shell) install
# unfree packages without extra flags. Does NOT cover flake-style
# commands (nix profile/build/shell/develop) -- those run in pure
# evaluation mode, which blocks reading env vars regardless of this
# setting, so `--impure` is still required alongside this for e.g.
# `nix profile add --impure nixpkgs#vscode`.
ENV NIXPKGS_ALLOW_UNFREE=1

# Best-effort GL driver dispatch: /run/opengl-driver is the path
# libglvnd/Mesa expect, normally set up by NixOS's module system.
# This is a known-incomplete area for non-NixOS Nix (see
# https://github.com/NixOS/nixpkgs/issues/62169, still open) -- this
# symlink alone may not be sufficient for hardware-accelerated GL to
# actually work; flagging as unverified rather than solved.
RUN ln -sf /nix/var/nix/profiles/default /run/opengl-driver

# Timezone default
# NOTE: no /usr/share/zoneinfo here -- nothing lands at FHS paths in
# this image. tzdata's zoneinfo is symlinked into the Nix profile
# tree instead, at /nix/var/nix/profiles/default/share/zoneinfo.
RUN ln -sf /nix/var/nix/profiles/default/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone
  
# Ensure a python3 binary is resolvable (Wolfi ships versioned
# python3.X only, no unversioned symlink)
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Install host-spawn (static binary, amd64/arm64) for host D-Bus/session integration.
COPY images/scripts/install-host-spawn.sh /tmp/install-host-spawn.sh
RUN sh /tmp/install-host-spawn.sh

# Static-binary backstop for fish/nushell where the distro's own package
# manager doesn't carry them (checked via command -v inside the script).
COPY images/scripts/install-shell.sh /tmp/install-shell.sh
RUN sh /tmp/install-shell.sh fish nu

# Silence fish/nushell startup & welcome messages (see silence-shell.sh).
COPY images/scripts/silence-shell.sh /tmp/silence-shell.sh
RUN sh /tmp/silence-shell.sh

# Install starship (static musl binary, amd64/arm64) into
# /usr/local/bin. Official images only.
COPY images/scripts/install-starship.sh /tmp/install-starship.sh
RUN sh /tmp/install-starship.sh

# Ship the default otter starship integration, read via $STARSHIP_CONFIG set below.
# Users can override it any time with their own $STARSHIP_CONFIG or ~/.config/starship.toml.
COPY images/scripts/starship.toml /usr/lib/otter/helpers/starship.toml
ENV STARSHIP_CONFIG=/usr/lib/otter/helpers/starship.toml