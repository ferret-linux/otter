# setup_slackpkg will upgrade or setup all packages for slackware based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_slackpkg()
{
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		yes | slackpkg upgrade-all -default_answer=yes -batch=yes
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		slackpkg update

		# Check if shell_bin is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user. Neither fish nor nu
		# is in Slackware's official repos -- the build-time
		# install-shell.sh static-binary backstop covers those instead.
		case "${shell_bin}" in
			fish | nu) shell_bin="bash" ;;
			*)
				if ! yes | slackpkg install -default_answer=yes -batch=yes "${shell_bin}"; then
					shell_bin="bash"
				fi
				;;
		esac
		shell_pkg="${shell_bin}"
		deps="
			${shell_pkg}
			bc
			xz
			atk
			glu
			mtr
			orc
			sbc
			srt
			tar
			zix
			zsh
			git
			curl
			perl
			zstd
			bash
			cups
			dbus
			faac
			fftw
			fish
			flac
			glew
			gzip
			lame
			less
			lilv
			llvm
			lsof
			mesa
			opus
			sdl2
			serd
			sord
			sudo
			suil
			time
			tree
			x264
			x265
			aften
			bluez
			bzip2
			cairo
			dav1d
			faad2
			glibc
			gnupg
			icu4c
			Imath
			lcms2
			libva
			pango
			pcre2
			rsync
			speex
			which
			xauth
			dcron
			wget2
			brotli
			c-ares
			libpsl
			ngtcp2
			a52dec
			dialog
			ffmpeg
			gnutls
			libaio
			libass
			libcap
			libdca
			libdrm
			libffi
			libgcc
			libogg
			libvpx
			libX11
			libXau
			libxcb
			man-db
			mpg123
			nettle
			pixman
			shadow
			sqlite
			sratom
			taglib
			libidn2
			libssh2
			nghttp2
			nghttp3
			fribidi
			infozip
			iputils
			libcdio
			libexif
			liblrdf
			libnice
			librsvg
			libtiff
			libXext
			libxml2
			ncurses
			openexr
			openssh
			python3
			tcpdump
			twolame
			wayland
			alsa-lib
			freetype
			graphene
			harfbuzz
			hostname
			iproute2
			keyutils
			libdecor
			libepoxy
			libglvnd
			libgudev
			libinput
			libvte-2
			libXdmcp
			nss-mdns
			opusfile
			pinentry
			pipewire
			qrencode
			xcb-util
			xvidcore
			xdg-utils
			dbus-glib
			diffutils
			findutils
			graphite2
			gstreamer
			json-glib
			libgcrypt
			libnotify
			libsecret
			libunwind
			libvisual
			libvorbis
			libXfixes
			procps-ng
			xorgproto
			zxing-cpp
			fluidsynth
			fontconfig
			gdk-pixbuf
			libfdk-aac
			libmodplug
			libplacebo
			libsndfile
			libxcb-glx
			libXdamage
			libXxf86vm
			soundtouch
			util-linux
			vulkan-sdk
            cyrus-sasl
			chromaprint
			libunibreak
			libxcb-dri2
			libxcb-dri3
			libxcb-sync
			wireplumber
			xcb-util-wm
			xisxwayland
			alsa-plugins
			at-spi2-core
			libgpg-error
			libpciaccess
			libunistring
			libxcb-randr
			libxcb-shape
			libxkbcommon
			libxshmfence
			opencore-amr
			xcb-util-xrm
			xdg-user-dirs
			vulkan-loader
			libjpeg-turbo
			libsamplerate
			libxcb-render
			libxcb-xfixes
			pipewire-jack
			wayland-utils
			pipewire-pulse
			glibc-zoneinfo
			libxcb-present
			xcb-util-image
            ca-certificates
			bash-completion
			gst-plugins-bad
			libdisplay-info
			xcb-util-cursor
			xcb-util-errors
			gst-plugins-base
			gst-plugins-good
			gst-plugins-ugly
			libcdio-paranoia
			shared-mime-info
			xcb-util-keysyms
			xkeyboard-config
			gst-plugins-libav
			wayland-protocols
			hicolor-icon-theme
			xdg-desktop-portal
			xcb-util-renderutil
			xorg-server-xwayland
		"

		# slackpkg install already no-ops safely on a pattern that matches
		# nothing ("No packages match the pattern for install."), so unlike
		# distrobox's setup_slackpkg() we don't pre-validate each package
		# with a "slackpkg search" loop here - see images/scripts/pkg-validator.sh
		# for the same reasoning applied at build time.
		rm -f /var/lock/slackpkg.*
		# shellcheck disable=SC2086
		yes | slackpkg install -default_answer=yes -batch=yes ${deps}
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	# shellcheck disable=SC2154 # HOST_LOCALE, HOST_LOCALE_LANG, HOST_LOCALE_ENCODING assigned by otter-init before sourcing this file
	if ! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"; then
		LANG="${HOST_LOCALE}" localedef -i "${HOST_LOCALE_LANG}" -f "${HOST_LOCALE_ENCODING}" "${HOST_LOCALE}" || true
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		yes | slackpkg install -default_answer=yes -batch=yes \
			${container_additional_packages}
	fi
}
