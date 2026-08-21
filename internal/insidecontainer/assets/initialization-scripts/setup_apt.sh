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
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${init}" -eq 0 ]; then
		# Loop through all the environment vars
		# and export them to the container.
		mkdir -p /etc/dpkg/dpkg.cfg.d/
		printf "" > /etc/dpkg/dpkg.cfg.d/00_otter
		# shellcheck disable=SC2154 # HOST_MOUNTS_RO, HOST_MOUNTS assigned by otter-init before sourcing this file
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
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
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
			echo "ttf-mscorefonts-installer msttcorefonts/accepted-mscorefonts-eula select true" | debconf-set-selections
			distro_deps="
				fish
				gstreamer1.0-libav
				gstreamer1.0-pipewire
				gstreamer1.0-plugins-bad
				gstreamer1.0-plugins-base
				gstreamer1.0-plugins-good
				gstreamer1.0-plugins-ugly
				libva-drm2
				libva-wayland2
				libva-x11-2
				libva2
				mesa-va-drivers
				systemd
				ubuntu-restricted-addons
				ubuntu-restricted-extras
				va-driver-all
				zsh
			"
			;;
		debian)
			distro_deps="
				ffmpeg
				fish
				gstreamer1.0-libav
				gstreamer1.0-pipewire
				gstreamer1.0-plugins-bad
				gstreamer1.0-plugins-base
				gstreamer1.0-plugins-good
				gstreamer1.0-plugins-ugly
				gstreamer1.0-tools
				intel-media-va-driver-non-free
				libavcodec-extra
				libflac12
				libgl1-mesa-dri
				libgl1-mesa-glx
				libmp3lame0
				libopus0
				libpipewire-0.3-0
				libvdpau1
				libvpx9
				libwayland-client0
				libx264-164
				libx265-199
				mesa-va-drivers
				pipewire
				pipewire-alsa
				pipewire-pulse
				systemd
				unrar
				wireplumber
				xdg-desktop-portal
				xwayland
				zsh
			"
			;;
		devuan)
			distro_deps="
				ffmpeg
				fish
				gstreamer1.0-libav
				gstreamer1.0-pipewire
				gstreamer1.0-plugins-bad
				gstreamer1.0-plugins-base
				gstreamer1.0-plugins-good
				gstreamer1.0-plugins-ugly
				gstreamer1.0-tools
				intel-media-va-driver-non-free
				libavcodec-extra
				libflac12
				libgl1-mesa-dri
				libgl1-mesa-glx
				libmp3lame0
				libopus0
				libpipewire-0.3-0
				libvdpau1
				libvpx9
				libx264-164
				libx265-199
				mesa-va-drivers
				pipewire
				pipewire-alsa
				pipewire-pulse
				sysvinit
				unrar
				wireplumber
				xdg-desktop-portal
				xwayland
				zsh
			"
			;;
		kali)
			distro_deps="
				bash
				ffmpeg
				fish
				gstreamer1.0-libav
				gstreamer1.0-plugins-bad
				gstreamer1.0-plugins-base
				gstreamer1.0-plugins-good
				gstreamer1.0-plugins-ugly
				gstreamer1.0-tools
				intel-media-va-driver-non-free
				kali-linux-core
				kali-linux-headless
				kali-tools-top10
				libavcodec-extra
				libdrm2
				libflac12
				libgl1-mesa-dri
				libmp3lame0
				libopus0
				libpipewire-0.3-0
				libva-drm2
				libva2
				libvdpau1
				libvpx9
				libwayland-client0
				libwayland-cursor0
				libwayland-egl1
				libwayland-server0
				libx11-6
				libx264-164
				libx265-199
				libxcb1
				libxcomposite1
				libxcursor1
				libxdamage1
				libxext6
				libxfixes3
				libxinerama1
				libxkbcommon-x11-0
				libxkbcommon0
				libxrandr2
				libxrender1
				mesa-va-drivers
				pipewire
				pipewire-alsa
				pipewire-jack
				pipewire-pulse
				systemd
				unrar
				wayland-protocols
				wireplumber
				xdg-desktop-portal
				xwayland
				zsh
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
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
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
