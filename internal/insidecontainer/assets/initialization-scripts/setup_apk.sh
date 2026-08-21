# setup_apk will upgrade or setup all packages for apk based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_apk()
{
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		apk update
		apk upgrade
		exit
	fi
	apk update
	# Sync base packages (e.g. musl) to the just-refreshed repo index before
	# installing anything new — an outdated base can cause newly installed
	# binaries to fail resolving newer symbols (e.g. renameat2).
	apk upgrade -Ua
	# Check if shell_pkg is available in distro's repo. If not we
	# fall back to bash, and we set the SHELL variable to bash so
	# that it is set up correctly for the user.
	if [ ! -f /usr/lib/otter/container.official ]; then
		if ! apk add "${shell_pkg}"; then
			shell_pkg="bash"
		fi
	fi
	distro_id="$(get_distro_id)"
	case "${distro_id}" in
		alpine)
			apk add alpine-base
			deps="
				alsa-lib
				coreutils
				desktop-file-utils
				docs
				ffmpeg
				fish
				fontconfig
				gcompat
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				hicolor-icon-theme
				libc-utils
				lsof
				man-pages
				mandoc
				mesa-egl
				mesa-gl
				musl-utils
				openrc
				openssh-client-default
				pinentry
				pipewire
				pipewire-jack
				pipewire-pulse
				shared-mime-info
				tar
				vpl-gpu-rt
				vte3
				wayland
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xwayland
				zsh
				$(apk search -q mesa-dri)
				$(apk search -q mesa-vulkan)
			"
			;;
		wolfi)
			apk add wolfi-base
			deps="
				alsa-lib
				alsa-utils
				busybox
				coreutils
				ffmpeg
				fish
				gnutar
				gst-libav
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				gstreamer
				gstreamer-pipewire
				libdrm
				man-db
				mesa
				openssh-client
				pinentry
				pipewire
				pipewire-jack
				pipewire-pulse
				posix-libc-utils
				script
				vpl-gpu-rt
				wayland
				wireplumber
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xwayland
				zsh
			"
			;;
		chimera)
			apk add base-bootstrap
			apk upgrade -Ua
			apk add chimera-repo-user
			apk update
			deps="
				base-full-man
				bc-gh
				dinit
				dinit-chimera
				ffmpeg
				fish
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				gtar
				libarchive-progs
				libcap-progs
				libva
				lsof
				mesa-dri
				mesa-vulkan
				ncurses-term
				opendoas
				openssh
				pipewire
				pipewire-alsa
				pipewire-gstreamer
				pipewire-jack
				pipewire-pulse
				util-linux-mount
				vpl-gpu-rt
				vte
				vulkan-loader
				wayland
				wget2
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xwayland
				zsh
			"
			;;
		*)
			printf "Error: unsupported apk-based distro '%s'.\n" "${distro_id}"
			printf "Error: could not set up base dependencies.\n"
			exit 127
			;;
	esac
	deps="${deps}
		${shell_pkg}
		bash
		bash-completion
		bc
		bzip2
		curl
		diffutils
		findmnt
		findutils
		gnupg
		gpg
		iproute2
		iputils
		keyutils
		less
		libcap
		mount
		ncurses
		ncurses-terminfo
		net-tools
		pigz
		python3
		rsync
		shadow
		sudo
		tcpdump
		tree
		tzdata
		umount
		unzip
		util-linux
		util-linux-login
		util-linux-misc
		vulkan-loader
		wget
		which
		xauth
		xz
		zip
		$(apk search -qe procps)
	"
	# shellcheck disable=SC2086
	found_deps="$(apk search -qe ${deps} | tr '\n' ' ')"
	install_pkg=""
	for dep in ${deps}; do
		# shellcheck disable=SC2249
		case " ${found_deps} " in
			*" ${dep} "*)
				install_pkg="${install_pkg} ${dep}"
				;;
		esac
	done
	# shellcheck disable=SC2086
	apk add --force-overwrite ${install_pkg}

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		apk del tzdata
		apk add tzdata
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		apk add --force-overwrite ${container_additional_packages}
	fi
}
