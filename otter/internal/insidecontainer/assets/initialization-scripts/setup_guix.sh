# setup_guix will upgrade or setup all packages for guix based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_guix()
{
	# If we need to upgrade, do it and exit, no further action required.
	# Upgrades installed packages against whatever channel is already
	# pulled -- `guix pull` (which upgrades Guix itself, its package
	# definitions and the daemon) is left to the user to run explicitly,
	# same as the Nix image leaves `nix flake update` to the user.
	# --profile is pinned explicitly to the target user's own profile,
	# matching Guix's own per-user convention (its installer's
	# /etc/profile.d/guix.sh sources $HOME/.guix-profile at each user's own
	# login) -- this call runs as root, with root's own $HOME now correctly
	# pointing at /root, so leaving --profile unset here would silently
	# install into /root/.guix-profile instead, invisible to the target
	# user (and unreachable regardless, since /root is chmod 700 on every
	# otter image). Ownership is fixed up afterward since guix, run as
	# root, otherwise leaves these paths root-owned inside a directory tree
	# the target user otherwise owns.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		guix upgrade --profile="${container_user_home}/.guix-profile"
		chown -R "${container_user_uid}:${container_user_gid}" "${container_user_home}/.guix-profile" \
			"${container_user_home}/.cache/guix" 2> /dev/null || :
		exit
	fi

	# No official Guix image exists to compare against here (there is no
	# upstream ghcr.io/nixos/nix-style image for Guix), and a foreign,
	# non-official image doesn't need any extra setup beyond what
	# guix.Containerfile / guix-install.sh already did at build time --
	# unlike Nix, which needs flakes enabled and a declarative package
	# set applied by hand on a foreign base. Informational only.
	if [ ! -f /usr/lib/otter/container.official ]; then
		printf "Info: guix does not require additional setup on non-official images.\n"
	fi

	# Install additional packages passed at otter-create time. Added one at
	# a time, rather than a single `guix install` call, so a single bad or
	# unresolvable package name doesn't abort the whole run under `set -o
	# errexit` -- warn and continue instead.
	if [ -n "${container_additional_packages}" ]; then
		for pkg in ${container_additional_packages}; do
			if ! guix install --profile="${container_user_home}/.guix-profile" "${pkg}"; then
				printf "Warning: failed to install additional package '%s'.\n" "${pkg}"
			fi
		done
		chown -R "${container_user_uid}:${container_user_gid}" "${container_user_home}/.guix-profile" \
			"${container_user_home}/.cache/guix" 2> /dev/null || :
	fi
}
