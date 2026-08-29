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
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		guix upgrade
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
			if ! guix install "${pkg}"; then
				printf "Warning: failed to install additional package '%s'.\n" "${pkg}"
			fi
		done
	fi
}