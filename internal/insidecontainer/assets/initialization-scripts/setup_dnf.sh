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
	# In dnf family official images, dnf is configured to ignore locale and docs
	# This however, results in a rather poor out-of-the-box experience
	# So, let's enable them.
	[ -e /etc/dnf/dnf.conf ] && sed -i '/tsflags=nodocs/d' /etc/dnf/dnf.conf
	[ -e /etc/yum.conf ] && sed -i '/tsflags=nodocs/d' /etc/yum.conf

	# Check if shell_pkg is available in distro's repo. If not we
	# fall back to bash, and we set the SHELL variable to bash so
	# that it is set up correctly for the user.
	if [ ! -f /usr/lib/otter/container.official ]; then
		if ! ${manager} install -y "${shell_pkg}" 2> /dev/null; then
			shell_pkg="bash"
		fi
	fi
	flags=""
	if [ "${manager}" = "dnf" ]; then
		flags="--allowerasing"
	fi
	distro_id="$(get_distro_id)"
	case "${distro_id}" in
		fedora)
			${manager} install -y \
				"https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm" \
				"https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm" 2> /dev/null || :
			${manager} config-manager setopt fedora-cisco-openh264.enabled=1 2> /dev/null || :
			distro_deps="
				fdk-aac
				ffmpeg
				ffmpeg-libs
				fish
				gstreamer1-plugins-bad-freeworld
				gstreamer1-plugins-base
				gstreamer1-plugins-good
				gstreamer1-plugins-ugly
				libaacs
				libavcodec-freeworld
				libheif-freeworld
				mesa-freeworld
				mesa-va-drivers-freeworld
				mesa-vulkan-drivers-freeworld
				pipewire
				pipewire-alsa
				pipewire-codec-aptx
				pipewire-gstreamer
				pipewire-jack-audio-connection-kit
				pipewire-pulseaudio
				systemd
				xorg-x11-server-Xwayland
				xpra-codecs-freeworld
				zsh
			"
			;;
		alma | centos | ol | rocky)
			${manager} install -y --nogpgcheck "https://dl.fedoraproject.org/pub/epel/epel-release-latest-$(rpm -E %rhel).noarch.rpm" 2> /dev/null || :
			${manager} install -y \
				"https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm" \
				"https://mirrors.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm" 2> /dev/null || :
			distro_deps="
				fdk-aac
				ffmpeg
				ffmpeg-libs
				fish
				gstreamer1-plugins-bad-freeworld
				gstreamer1-plugins-base
				gstreamer1-plugins-good
				gstreamer1-plugins-ugly
				libaacs
				libavcodec-freeworld
				libheif-freeworld
				mesa-freeworld
				mesa-va-drivers-freeworld
				mesa-vulkan-drivers-freeworld
				pipewire
				pipewire-alsa
				pipewire-codec-aptx
				pipewire-gstreamer
				pipewire-jack-audio-connection-kit
				pipewire-pulseaudio
				systemd
				xorg-x11-server-Xwayland
				xpra-codecs-freeworld
				zsh
			"
			;;
		rhel)
			distro_deps="
				fish
				gstreamer1
				lame
				opus
				pipewire
				pipewire-pulseaudio
				systemd
				wireplumber
				x264
				x265
				zsh
			"
			;;
		amzn)
			${manager} install -y spal-release
			distro_deps="
				alsa-lib
				desktop-file-utils
				ffmpeg-free
				fish
				fontconfig
				gstreamer1
				gstreamer1-plugins-bad-free
				gstreamer1-plugins-base
				gstreamer1-plugins-good
				hicolor-icon-theme
				libva
				libvdpau
				libX11
				libXcomposite
				libXcursor
				libXdamage
				libXext
				libXfixes
				libXi
				libXinerama
				libxcb
				libxkbcommon
				libxkbcommon-x11
				libXrandr
				libXrender
				mesa-libEGL
				mesa-libGL
				mesa-libgbm
				mesa-va-drivers
				mesa-vdpau-drivers
				pipewire-alsa
				pipewire-gstreamer
				pipewire-jack-audio-connection-kit
				pipewire-pulseaudio
				shared-mime-info
				systemd
				vulkan-loader
				wayland
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xorg-x11-server-Xwayland
				zsh
			"
			;;
		*)
			printf "Error: unsupported dnf-based distro '%s'.\n" "${distro_id}"
			printf "Error: could not set up base dependencies.\n"
			exit 127
			;;
	esac
	deps="${distro_deps}
		${shell_pkg}
		bash-completion
		bc
		bzip2
		cracklib-dicts
		curl
		diffutils
		dnf-plugins-core
		findutils
		glibc-all-langpacks
		glibc-common
		glibc-locale-source
		gnupg2
		gnupg2-smime
		hostname
		iproute
		iputils
		keyutils
		krb5-libs
		less
		lsof
		man-db
		man-pages
		mtr
		ncurses
		nss-mdns
		openssh-clients
		pam
		passwd
		pigz
		pinentry
		procps-ng
		python3
		rsync
		shadow-utils
		sudo
		tcpdump
		time
		traceroute
		tree
		tzdata
		unzip
		util-linux
		util-linux-script
		vte-profile
		wget
		wget2-wget
		which
		whois
		words
		xorg-x11-xauth
		xz
		zip
		mesa-dri-drivers
		mesa-vulkan-drivers
		vulkan
	"
	# shellcheck disable=SC2086,2046,2248
	${manager} install ${flags} -y $(${manager} list -q ${deps} |
		grep -v "Packages" |
		grep "$(uname -m)" |
		cut -d' ' -f1)

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
