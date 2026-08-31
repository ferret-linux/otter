# setup_xbps_exceptions will create path-excludes for host mounts, xbps only.
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
setup_xbps_exceptions()
{
	# We have to lock this paths from xbps extraction, as it's incompatible with otter's
	# mount process.
	cat << EOF > /etc/xbps.d/otter-ignore.conf
noextract=/etc/passwd
noextract=/etc/hosts
noextract=/etc/host.conf
noextract=/etc/hostname
noextract=/etc/localtime
noextract=/etc/machine-id
noextract=/etc/resolv.conf
EOF
}

# setup_xbps will upgrade or setup all packages for xbps based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_xbps()
{
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		xbps-install -Syu
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# Ensure we avoid errors by keeping xbps updated
		xbps-install -Syu xbps

		# Check if shell_bin is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user. xbps names fish and
		# nushell differently from their binary names.
		case "${shell_bin}" in
			fish) shell_pkg="fish-shell" ;;
			nu) shell_pkg="nushell" ;;
			*) shell_pkg="${shell_bin}" ;;
		esac
		if ! xbps-install -Sy "${shell_pkg}"; then
			shell_bin="bash"
		fi
		xbps-install -Sy void-repo-nonfree
		deps="
			bc
			xz
			mtr
			nss
			zip
			zsh
			curl
			lame
			less
			lsof
			opus
			pigz
			sudo
			time
			tree
			vte3
			wget
			x265
			bzip2
			rsync
			runit
			unzip
			which
			xauth
			ffmpeg
			gnupg2
			libdrm
			libvpx
			man-db
			shadow
			tzdata
			libflac
			libx264
			ncurses
			openssh
			python3
			wayland
			alsa-lib
			iproute2
			mesa-dri
			mit-krb5
			pinentry
			pipewire
			diffutils
			findutils
			gst-libav
			gstreamer
			inetutils
			procps-ng
			xdg-utils
			alsa-utils
			traceroute
			util-linux
			wireplumber
			${shell_pkg}
			pinentry-tty
			mit-krb5-libs
			vulkan-loader
			alsa-pipewire
			xdg-user-dirs
			pipewire-pulse
			bash-completion
			gst-plugins-bad
			mit-krb5-client
			gst-plugins-base
			gst-plugins-good
			gst-plugins-ugly
			mesa-vulkan-intel
			xdg-desktop-portal
			mesa-vulkan-radeon
			gstreamer1-pipewire
			xorg-server-xwayland
		"
		# shellcheck disable=SC2086,2046
		xbps-install -Sy $(xbps-query -Rs '*' | awk '{print $2}' | sed 's/-[^-]*$//' | grep -E "^($(echo ${deps} | tr ' ' '|'))$")
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if command -v locale && {
		! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"
	}; then
		sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/default/libc-locales
		sed -i "s|#.*${HOST_LOCALE}|${HOST_LOCALE}|g" /etc/default/libc-locales
		xbps-reconfigure --force glibc-locales
	fi

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		xbps-install --force -y tzdata
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		xbps-install -Sy ${container_additional_packages}
	fi
}
