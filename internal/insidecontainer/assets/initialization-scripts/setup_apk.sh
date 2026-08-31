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
	if [ ! -f /usr/lib/otter/container.official ]; then
		apk update
		# Sync base packages (e.g. musl) to the just-refreshed repo index before
		# installing anything new — an outdated base can cause newly installed
		# binaries to fail resolving newer symbols (e.g. renameat2).
		apk upgrade -Ua
		# Check if shell_bin is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user. nu's apk package is
		# named nushell, not nu.
		case "${shell_bin}" in
			nu) shell_pkg="nushell" ;;
			*) shell_pkg="${shell_bin}" ;;
		esac
		if ! apk add "${shell_pkg}"; then
			shell_bin="bash"
		fi
		distro_id="$(get_distro_id)"
		case "${distro_id}" in
			alpine)
				apk add alpine-base
				deps="
					tar
					docs
					lsof
					vte3
					mandoc
					openrc
					gcompat
					mesa-gl
					alsa-lib
					mesa-egl
					pinentry
					coreutils
					man-pages
					fontconfig
					libc-utils
					musl-utils
					shared-mime-info
					desktop-file-utils
					hicolor-icon-theme
					openssh-client-default
					$(apk search -q mesa-dri)
					$(apk search -q mesa-vulkan)
				"
				;;
			wolfi)
				apk add wolfi-base
				deps="
					mesa
					gnutar
					libdrm
					man-db
					script
					busybox
					alsa-lib
					pinentry
					coreutils
					gst-libav
					gstreamer
					alsa-utils
					wireplumber
					openssh-client
					posix-libc-utils
					gstreamer-pipewire
				"
				;;
			chimera)
				apk add base-bootstrap
				apk upgrade -Ua
				apk add chimera-repo-user
				apk update
				deps="
					vte
					gtar
					lsof
					bc-gh
					dinit
					libva
					wget2
					openssh
					mesa-dri
					opendoas
					mesa-vulkan
					libcap-progs
					ncurses-term
					base-full-man
					dinit-chimera
					libarchive-progs
					util-linux-mount
					pipewire-gstreamer
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
			bc
			xz
			gpg
			zip
			zsh
			bash
			curl
			fish
			less
			pigz
			sudo
			tree
			wget
			bzip2
			gnupg
			mount
			rsync
			unzip
			which
			xauth
			ffmpeg
			libcap
			shadow
			tzdata
			umount
			findmnt
			iputils
			ncurses
			python3
			tcpdump
			wayland
			iproute2
			keyutils
			pipewire
			xwayland
			diffutils
			findutils
			net-tools
			xdg-utils
			util-linux
			vpl-gpu-rt
			pipewire-alsa
			pipewire-jack
			vulkan-loader
			xdg-user-dirs
			pipewire-pulse
			bash-completion
			gst-plugins-bad
			util-linux-misc
			gst-plugins-base
			gst-plugins-good
			gst-plugins-ugly
			ncurses-terminfo
			util-linux-login
			xdg-desktop-portal
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
	fi

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
