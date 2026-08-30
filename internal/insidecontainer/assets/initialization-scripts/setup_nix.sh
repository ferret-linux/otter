# setup_nix will upgrade or setup all packages for nix based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_nix()
{
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		# --profile pins this to the same shared profile every other nix
		# profile call in this file targets explicitly. Without it, this
		# resolves the default $HOME/.nix-profile instead -- which, being
		# root-invoked, gets auto-created pointing at this same profile on a
		# container that's never had it touched directly, but which instead
		# resolves to the target user's own personal profile (if the target
		# user has ever run a bare `nix profile install` themselves) and
		# upgrades that one instead, as root.
		nix profile upgrade \
			--profile /nix/var/nix/profiles/default \
			'.*'
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# Flakes / the `nix` CLI are not enabled by default on a foreign nix
		# image. Rebuilt from scratch rather than appended, pulling forward
		# only trusted-public-keys from the image's original nix.conf (that
		# key is image-specific and shouldn't be hardcoded here);
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
		# Both settings blocks below are written in one go via a
		# read-modify-write against a fresh /tmp file rather than appending
		# directly to /etc/nix/nix.conf: on this image, that path is a
		# symlink into the Nix store, and shell `>>` redirection follows
		# symlinks, so an in-place append would silently write into an
		# immutable store path (root bypasses the store's read-only file
		# perms) and corrupt it -- `rm -f` on a symlink only removes the
		# symlink itself, never the store target, which is what makes this
		# safe.
		grep '^trusted-public-keys' /etc/nix/nix.conf > /tmp/nix.conf.new
		printf '%s\n' \
			'sandbox = false' \
			'keep-outputs = true' \
			'filter-syscalls = false' \
			'keep-derivations = true' \
			'auto-optimise-store = true' \
			'build-users-group = nixbld' \
			'experimental-features = nix-command flakes' \
			>> /tmp/nix.conf.new
		rm -f /etc/nix/nix.conf
		mv /tmp/nix.conf.new /etc/nix/nix.conf

		# Pin the "nixpkgs" flake registry entry explicitly to
		# nixpkgs-unstable, so every `nixpkgs#pkg` reference below (and
		# anything a user runs later) resolves deterministically instead of
		# whatever the global flake registry's default happens to be.
		# Fallback images are unstable-only for now; a stable channel option
		# may be added later.
		nix registry pin nixpkgs github:NixOS/nixpkgs/nixpkgs-unstable

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
		# All five files are kept under /etc/otter so users can inspect and
		# adjust the package set after setup.
		#
		# nix-settings.nix holds the nixpkgs `config` attrset (currently
		# just allowUnfree) applied when packages-lib.nix instantiates pkgs.
		#
		# packages-lib.nix is a shared helper imported by all five tiers
		# below: it centralizes the `pkgs` binding (instantiated via
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
		mkdir -p /etc/otter
		cat > /etc/otter/nix-settings.nix <<'EOF'
{
  allowUnfree = true;
}
EOF

		cat > /etc/otter/packages-lib.nix <<'EOF'
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

		cat > /etc/otter/packages-core.nix <<'EOF'
let
  common = import /etc/otter/packages-lib.nix;
in
with common.pkgs;

common.onlySupported [
  bc
  lsof
  sudo
  tree
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

		cat > /etc/otter/packages-shell.nix <<'EOF'
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

		cat > /etc/otter/packages-multimedia.nix <<'EOF'
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

		cat > /etc/otter/packages-network.nix <<'EOF'
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
  coreutils-full
]
EOF

		cat > /etc/otter/packages-utilities.nix <<'EOF'
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
		# then install the five-tier declarative set in the same step so
		# there's no window where coreutils/findutils are missing but a
		# later step still needs them. Deliberately NOT removed: nix,
		# nss-cacert, iana-etc, gnugrep, gzip -- not redeclared below, and
		# nix/nss-cacert are load-bearing for the nix CLI itself.
		nix profile remove \
			--profile /nix/var/nix/profiles/default \
			bash-interactive coreutils-full curl findutils gnutar \
			less man-db openssh wget which git-minimal
		nix profile add \
			--profile /nix/var/nix/profiles/default \
			--priority 0 \
			--file /etc/otter/packages-core.nix
		nix profile add \
			--profile /nix/var/nix/profiles/default \
			--priority 1 \
			--file /etc/otter/packages-shell.nix
		nix profile add \
			--profile /nix/var/nix/profiles/default \
			--priority 2 \
			--file /etc/otter/packages-multimedia.nix
		nix profile add \
			--profile /nix/var/nix/profiles/default \
			--priority 3 \
			--file /etc/otter/packages-network.nix
		nix profile add \
			--profile /nix/var/nix/profiles/default \
			--priority 4 \
			--file /etc/otter/packages-utilities.nix

		# This image has no NixOS module system to generate PAM service
		# files or /etc/login.defs the way a real NixOS install would, so
		# shadow's password tools (passwd, chpasswd) have no policy to
		# authenticate against and fail outright. shadow's own PAM linkage
		# is left enabled (not nulled out) so this matches every other
		# image, which all get working PAM-backed passwd/chpasswd from
		# their distro's package manager. pam_unix.so is referenced by
		# bare name, same as the existing pam-su template (otter-init) --
		# this resolves correctly against this build's own Nix store path
		# because libpam compiles its module search path in at build time
		# (DEFAULT_MODULE_PATH), rather than reading it from the config
		# file, so no store path needs to be hardcoded here.
		#
		# ENCRYPT_METHOD is pinned to SHA512 explicitly, rather than
		# relying on whatever this shadow build's compiled-in default
		# resolves to, since it's supported unconditionally by every
		# libxcrypt build, with or without the optional yescrypt/bcrypt
		# modules this build also enables.
		mkdir -p /etc/pam.d
		printf '%s\n' \
			'password    required    pam_unix.so' \
			> /etc/pam.d/passwd
		printf '%s\n' \
			'password    required    pam_unix.so' \
			> /etc/pam.d/chpasswd
		printf '%s\n' \
			'ENCRYPT_METHOD SHA512' \
			> /etc/login.defs

		# otter-init looks for the systemd binary and unit files at the FHS
		# paths /usr/lib/systemd/systemd and /usr/lib/systemd/system/*, and
		# expects `systemctl` on $PATH. Nix profile-installs systemd's own
		# output tree under <profile>/lib/systemd instead of at those FHS
		# paths, so symlink the whole directory across in one shot rather
		# than each file individually.
		mkdir -p /usr/lib
		ln -sf /nix/var/nix/profiles/default/lib/systemd /usr/lib/systemd

		# Timezone default
		# NOTE: no /usr/share/zoneinfo here -- nothing lands at FHS paths in
		# this image. tzdata's zoneinfo is symlinked into the Nix profile
		# tree instead, at /nix/var/nix/profiles/default/share/zoneinfo.
		ln -sf /nix/var/nix/profiles/default/share/zoneinfo/UTC /etc/localtime
		echo "UTC" > /etc/timezone
	fi

	# glibc on non-NixOS systems can't assume an FHS locale-archive path, so
	# nixpkgs' glibc is patched to read from $LOCALE_ARCHIVE instead
	# (documented nixpkgs behavior, not a workaround specific to this
	# image). Without this, locale-dependent tools emit "cannot change
	# locale" warnings or misbehave. ENV bakes into an image at build time,
	# so a runtime script has to persist these another way -- dropped into
	# /etc/profile.d alongside otter's own profile scripts. Written
	# unconditionally (official image or not), same as the HOST_LOCALE
	# fixup in every other setup_*.sh (apt, dnf, pacman, slackpkg, zypper,
	# emerge) -- an official nix image bakes LANG=en_US.UTF-8 at build time
	# (nix.Containerfile), and this brings it in line with the host
	# regardless of that gate above.
	#
	# NIXPKGS_ALLOW_UNFREE lets classic-style commands (nix-env, nix-build,
	# nix-shell) install unfree packages without extra flags. Does NOT
	# cover flake-style commands (nix profile/build/shell/develop) -- those
	# run in pure evaluation mode, which blocks reading env vars regardless
	# of this setting, so `--impure` is still required alongside this for
	# e.g. `nix profile add --impure nixpkgs#vscode`.
	# shellcheck disable=SC2154 # HOST_LOCALE assigned by otter-init before sourcing this file
	cat > /etc/profile.d/otter_nix_env.sh <<EOF
export LANG=${HOST_LOCALE}
export LOCALE_ARCHIVE=/nix/var/nix/profiles/default/lib/locale/locale-archive
export NIXPKGS_ALLOW_UNFREE=1
EOF

	# Install additional packages passed at otter-create time. Added one at a
	# time, rather than in a single `nix profile add`, so a single bad or
	# colliding package name doesn't abort the whole run under `set -o
	# errexit` -- warn and continue instead. Priority -5 wins over the
	# curated tiers above (core is priority 0, and lower numbers take
	# precedence), since these were explicitly requested by the user.
	if [ -n "${container_additional_packages}" ]; then
		for pkg in ${container_additional_packages}; do
			if ! nix profile add \
				--profile /nix/var/nix/profiles/default \
				--priority -5 \
				"nixpkgs#${pkg}"; then
				printf "Warning: failed to install additional package '%s'.\n" "${pkg}"
			fi
		done
	fi
}
