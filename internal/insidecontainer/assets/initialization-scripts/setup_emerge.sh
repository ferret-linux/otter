# setup_emerge will upgrade or setup all packages for gentoo based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_emerge()
{
	# Check if the container we are using has a ::gentoo repo defined,
	# if it is defined and it is empty, then synchroznize it.
	gentoo_repo="$(portageq get_repo_path / gentoo)"
	if [ -n "${gentoo_repo}" ] && [ ! -e "${gentoo_repo}" ]; then
		emerge-webrsync
	fi
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		emerge --sync
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# Check if shell_pkg is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user.
		getuto 2>/dev/null || :
		# Portage packages live under app-shells/ -- this was previously
		# missing here (though present in the deps list below), and nu's
		# package is named nushell, not nu.
		case "${shell_bin}" in
			nu) shell_pkg="app-shells/nushell" ;;
			*) shell_pkg="app-shells/${shell_bin}" ;;
		esac
		if ! emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg "${shell_pkg}"; then
			shell_bin="bash"
		fi
		deps="
			sys-devel/bc
			app-arch/pigz
			app-text/tree
			net-misc/curl
			net-misc/wget
			sys-apps/less
			app-admin/sudo
			app-arch/unrar
			x11-apps/xauth
			x11-libs/libXi
			x11-libs/libdrm
			app-crypt/gnupg
			app-shells/fish
			dev-lang/python
			media-libs/flac
			media-libs/mesa
			media-libs/opus
			media-libs/tiff
			media-libs/x264
			media-libs/x265
			sys-apps/openrc
			sys-apps/shadow
			x11-libs/libX11
			x11-libs/libxcb
        	media-libs/dav1d
			media-libs/libva
			media-sound/lame
			sys-libs/ncurses
			sys-process/lsof
			x11-libs/libXext
			x11-libs/libvdpau
			media-libs/libaom
			media-libs/libpng
			media-libs/libvpx
			x11-base/xwayland
			x11-misc/xdg-utils
			app-crypt/pinentry
			media-libs/libexif
			media-libs/libheif
			media-libs/libwebp
			media-video/ffmpeg
			sys-apps/diffutils
			sys-apps/findutils
			sys-process/procps
			x11-libs/libXfixes
			x11-libs/libXrandr
			media-libs/libde265
			media-libs/openh264
			sys-apps/util-linux
			x11-libs/libXcursor
			x11-libs/libXdamage
			media-libs/gstreamer
			media-video/pipewire
			x11-libs/libXinerama
			x11-libs/libxkbcommon
			x11-misc/xdg-user-dirs
			app-portage/gentoolkit
			x11-libs/libXcomposite
			app-shells/${shell_pkg}
			media-video/wireplumber
			media-libs/libjpeg-turbo
			media-libs/vulkan-layers
			media-libs/vulkan-loader
			app-shells/bash-completion
			media-libs/gst-plugins-bad
			sys-apps/xdg-desktop-portal
			media-libs/gst-plugins-base
			media-libs/gst-plugins-good
			media-libs/gst-plugins-ugly
			media-plugins/gst-plugins-libav
		"
		install_pkg=""
		for dep in ${deps}; do
			if [ "$(emerge --ask=n --search "${dep}" | grep "Applications found" | grep -Eo "[0-9]")" -gt 0 ]; then
				# shellcheck disable=SC2086
				install_pkg="${install_pkg} ${dep}"
			fi
		done
		# shellcheck disable=SC2086
		emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg --keep-going ${install_pkg}
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if ! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"; then
		sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen
		sed -i "s|#.*${HOST_LOCALE}|${HOST_LOCALE}|g" /etc/locale.gen
		locale-gen
		cat << EOF > /etc/env.d/02locale
LANG=${HOST_LOCALE}
LC_CTYPE=${HOST_LOCALE}
EOF
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		emerge --ask=n --autounmask-continue --noreplace --quiet-build \
			--getbinpkg --keep-going ${container_additional_packages}
	fi
}
