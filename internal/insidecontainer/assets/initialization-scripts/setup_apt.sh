# setup_deb_exceptions will create path-excludes for host mounts, dpkg/apt only.
# Arguments:
#   None
# Expected global variables:
#   init: if this is an initful container
#   HOST_MOUNTS_RO: list of readonly mountpoints, to avoid
#   HOST_MOUNTS: list of readwrite mountpoints, to avoid
# Expected env variables:
#   None
# Outputs:
#   None
setup_deb_exceptions()
{
	setup_pkg_manager_hooks

	# In case of an DEB distro, we can specify that our bind_mount directories
	# have to be ignored. This prevents conflicts during package installations.
	if [ "${init}" -eq 0 ]; then
		# Loop through all the environment vars
		# and export them to the container.
		mkdir -p /etc/dpkg/dpkg.cfg.d/
		printf "" > /etc/dpkg/dpkg.cfg.d/00_otter
		for net_mount in ${HOST_MOUNTS_RO} ${HOST_MOUNTS}; do
			printf "path-exclude %s/*\n" "${net_mount}" >> /etc/dpkg/dpkg.cfg.d/00_otter
		done

		# Also we put a hook to clear some critical paths that do not play well
		# with read only filesystems, like Systemd.
		if [ -d "/etc/apt/apt.conf.d/" ]; then
			printf 'DPkg::Pre-Invoke {/etc/otter-pre-hook.sh};\n' > /etc/apt/apt.conf.d/00_otter
			printf 'DPkg::Post-Invoke {/etc/otter-post-hook.sh};\n' >> /etc/apt/apt.conf.d/00_otter
		fi
	fi
}

# setup_apt will upgrade or setup all packages for apt based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_apt()
{
	export DEBIAN_FRONTEND=noninteractive

	# If we need to upgrade, do it and exit, no further action required.
	if [ "${upgrade}" -ne 0 ]; then
		apt-get update
		apt-get upgrade -o Dpkg::Options::="--force-confold" -y
		exit
	fi
	# In Ubuntu official images, dpkg is configured to ignore locale and docs
	# This however, results in a rather poor out-of-the-box experience
	# So, let's enable them.
	rm -f /etc/dpkg/dpkg.cfg.d/excludes

	apt-get update
	# Check if shell_pkg is available in distro's repo. If not we
	# fall back to bash, and we set the SHELL variable to bash so
	# that it is set up correctly for the user.
	if [ ! -f /usr/lib/otter/container.official ]; then
		if ! apt-get install -y "${shell_pkg}"; then
			shell_pkg="bash"
		fi
	fi
	distro_id="$(get_distro_id)"
	case "${distro_id}" in
		ubuntu)
			distro_deps="
				systemd
			"
			;;
		debian)
			distro_deps="
				systemd
			"
			;;
		devuan)
			distro_deps="
				sysvinit
			"
			;;
		*)
			printf "Error: unsupported apt-based distro '%s'.\n" "${distro_id}"
			printf "Error: could not set up base dependencies.\n"
			exit 127
			;;
	esac
	deps="${distro_deps}
		${shell_pkg}
		apt-utils
		bash-completion
		bc
		bzip2
		curl
		dialog
		diffutils
		findutils
		gnupg
		gnupg2
		gpgsm
		hostname
		iproute2
		iputils-ping
		keyutils
		language-pack-en
		less
		libcap2-bin
		libkrb5-3
		libnss-mdns
		libnss-myhostname
		libvte-2.9*-common
		libvte-common
		locales
		lsof
		man-db
		manpages
		mtr
		ncurses-base
		openssh-client
		passwd
		pigz
		pinentry-curses
		procps
		python3
		rsync
		sudo
		tcpdump
		time
		traceroute
		tree
		tzdata
		unzip
		util-linux
		wget
		xauth
		xz-utils
		zip
		libgl1
		libegl-mesa0
		libegl1-mesa
		libgl1-mesa-glx
		libegl1
		libglx-mesa0
		libvulkan1
		mesa-vulkan-drivers
	"
	# shellcheck disable=SC2086,2046
	apt-get install -y $(apt-cache show ${deps} 2> /dev/null | grep "Package:" | sort -u | cut -d' ' -f2-)

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	if [ -e /etc/locale.gen ] && {
		! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"
	}; then
		sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen
		sed -i "s|#.*${HOST_LOCALE}|${HOST_LOCALE}|g" /etc/locale.gen
		locale-gen
		update-locale LC_ALL="${HOST_LOCALE}" LANG="${HOST_LOCALE}"
		dpkg-reconfigure locales
	fi

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		apt-get --reinstall install tzdata
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		apt-get install -y ${container_additional_packages}
	fi

	# Make APT return to normal so it works as intended for user
	unset DEBIAN_FRONTEND
}