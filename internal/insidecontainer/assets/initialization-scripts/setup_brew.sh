# setup_brew will upgrade or setup all packages for brew based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_brew()
{
	# If we need to upgrade, do it and exit, no further action required.
	# Unlike `guix upgrade`, `brew upgrade` alone only upgrades formulae
	# against whatever formula index is already cached locally -- it
	# won't see anything new until `brew update` refreshes that index
	# first, so both are needed here for an upgrade to actually upgrade.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		brew update
		brew upgrade
		exit
	fi

	# No official Homebrew-on-Linux image exists to compare against here,
	# and a foreign, non-official image doesn't need any extra setup
	# beyond what brew.Containerfile / the official installer already did
	# at build time -- unlike Nix, which needs flakes enabled and a
	# declarative package set applied by hand on a foreign base.
	# Informational only.
	if [ ! -f /usr/lib/otter/container.official ]; then
		printf "Info: brew does not require additional setup on non-official images.\n"
	fi

	# Install additional packages passed at otter-create time. Added one at
	# a time, rather than a single `brew install` call, so a single bad or
	# unresolvable formula name doesn't abort the whole run under `set -o
	# errexit` -- warn and continue instead.
	if [ -n "${container_additional_packages}" ]; then
		for pkg in ${container_additional_packages}; do
			if ! brew install "${pkg}"; then
				printf "Warning: failed to install additional package '%s'.\n" "${pkg}"
			fi
		done
	fi
}