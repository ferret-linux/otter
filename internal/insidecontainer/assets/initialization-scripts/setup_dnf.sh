# setup_dnf will upgrade or setup all packages for dnf based systems.
# Arguments:
#   manager: dnf
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_dnf()
{
	manager=$1

	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		${manager} upgrade -y
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# In dnf family official images, dnf is configured to ignore locale and docs
		# This however, results in a rather poor out-of-the-box experience
		# So, let's enable them.
		[ -e /etc/dnf/dnf.conf ] && sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf
		[ -e /etc/yum.conf ] && sed -i '/tsflags=nodocs/d' /etc/yum.conf

		# Check if shell_bin is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user. nu only ships an
		# official dnf package on Fedora itself (as "nu") -- everywhere
		# else on dnf (RHEL-family, Amazon) it isn't packaged at all, so
		# skip the doomed install attempt there.
		if [ "${shell_bin}" = "nu" ] && [ "$(get_distro_id)" != "fedora" ]; then
			shell_bin="bash"
		elif ! ${manager} install -y "${shell_bin}" 2> /dev/null; then
			shell_bin="bash"
		fi
		flags=""
		if [ "${manager}" = "dnf" ]; then
			flags="--allowerasing"
		fi
		distro_id="$(get_distro_id)"
		case "${distro_id}" in
			fedora | alma | centos | ol | rocky)
				if [ "${distro_id}" = "fedora" ]; then
					${manager} install -y \
						dnf5-plugins \
						"https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm" \
						"https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm" 2> /dev/null || :
					${manager} config-manager setopt fedora-cisco-openh264.enabled=1 2> /dev/null || :
				else
					${manager} install -y --nogpgcheck "https://dl.fedoraproject.org/pub/epel/epel-release-latest-$(rpm -E %rhel).noarch.rpm" 2> /dev/null || :
					${manager} install -y \
						"https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm" \
						"https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm" 2> /dev/null || :
				fi
				distro_deps="
					ffmpeg
					fdk-aac
					libaacs
					ffmpeg-libs
					mesa-freeworld
					libheif-freeworld
					pipewire-codec-aptx
					libavcodec-freeworld
					xpra-codecs-freeworld
					gstreamer1-plugins-base
					gstreamer1-plugins-good
					gstreamer1-plugins-ugly
					xorg-x11-server-Xwayland
					mesa-va-drivers-freeworld
					mesa-vulkan-drivers-freeworld
					gstreamer1-plugins-bad-freeworld
				"
				;;
			rhel)
			# RHEL doesnt support adding non-free packages unless you have active subscription
			# the UBI images have small repos by default so we are not gonna add all the packages
				distro_deps="
					lame
					opus
					x264
					x265
					gstreamer1
					wireplumber
				"
				;;
			amzn)
				${manager} install -y spal-release
				distro_deps="
					libXi
					libva
					libX11
					libxcb
					libXext
					wayland
					alsa-lib
					libvdpau
					libXfixes
					libXrandr
					fontconfig
					gstreamer1
					libXcursor
					libXdamage
					libXrender
					mesa-libGL
					ffmpeg-free
					libXinerama
					mesa-libEGL
					mesa-libgbm
					libxkbcommon
					libXcomposite
					vulkan-loader
					xdg-user-dirs
					mesa-va-drivers
					libxkbcommon-x11
					shared-mime-info
					desktop-file-utils
					hicolor-icon-theme
					mesa-vdpau-drivers
					gstreamer1-plugins-base
					gstreamer1-plugins-good
					xorg-x11-server-Xwayland
					gstreamer1-plugins-bad-free
				"
				;;
			*)
				printf "Error: unsupported dnf-based distro '%s'.\n" "${distro_id}"
				printf "Error: could not set up base dependencies.\n"
				exit 127
				;;
		esac
		deps="${distro_deps}
			bc
			xz
			mtr
			pam
			zip
			zsh
			curl
			fish
			less
			lsof
			pigz
			sudo
			time
			tree
			wget
			bzip2
			rsync
			unzip
			which
			whois
			words
			gnupg2
			man-db
			passwd
			tzdata
			vulkan
			iproute
			iputils
			ncurses
			python3
			systemd
			tcpdump
			hostname
			keyutils
			nss-mdns
			pinentry
			pipewire
			xdg-utils
			diffutils
			findutils
			krb5-libs
			man-pages
			procps-ng
			traceroute
			util-linux
			wget2-wget
			vte-profile
			${shell_pkg}
			glibc-common
			gnupg2-smime
			shadow-utils
			pipewire-alsa
			cracklib-dicts
			xorg-x11-xauth
			bash-completion
			openssh-clients
			dnf-plugins-core
			mesa-dri-drivers
			util-linux-script
			xdg-desktop-portal
			pipewire-gstreamer
			glibc-all-langpacks
			glibc-locale-source
			mesa-vulkan-drivers
			pipewire-pulseaudio
			pipewire-jack-audio-connection-kit
		"
		# shellcheck disable=SC2086,2046,2248
		${manager} install ${flags} -y $(${manager} list -q ${deps} |
			grep -v "Packages" |
			grep "$(uname -m)" |
			cut -d' ' -f1)
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	if [ ! -e /usr/share/i18n/charmaps ]; then
		${manager} reinstall -y glibc-common
	fi
	# shellcheck disable=SC2154 # HOST_LOCALE, HOST_LOCALE_LANG, HOST_LOCALE_ENCODING assigned by otter-init before sourcing this file
	if ! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"; then
		LANG="${HOST_LOCALE}" localedef -i "${HOST_LOCALE_LANG}" -f "${HOST_LOCALE_ENCODING}" "${HOST_LOCALE}"
	fi

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		${manager} reinstall -y tzdata
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		${manager} install -y ${container_additional_packages}
	fi
}
