# setup_zypper will upgrade or setup all packages for zypper based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_zypper()
{
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		zypper dup -y
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# nu isn't in openSUSE's default repos (it's only reachable via the
		# shells OBS repo) -- adding third-party repos is out of scope here,
		# so skip the doomed install attempt and fall back to bash directly.
		if [ "${shell_bin}" = "nu" ]; then
			shell_bin="bash"
		elif ! zypper install -y "${shell_bin}"; then
			shell_bin="bash"
		fi
		shell_pkg="${shell_bin}"

		# In openSUSE official images, zypper is configured to ignore recommended
		# packages (i.e., weak dependencies). This however, results in a rather
		# poor out-of-the-box experience (e.g., when trying to run GUI apps).
		# So, let's enable them. For the same reason, we make sure we install
		# docs.
		if [ -d /etc/zypp/zypp.conf.d ]; then
			cat << EOF > /etc/zypp/zypp.conf.d/99-otter.conf
[main]
solver.onlyRequires = false
rpm.install.excludedocs = no
EOF
		else
			sed -i 's/.*solver.onlyRequires.*/solver.onlyRequires = false/g' /etc/zypp/zypp.conf
			sed -i 's/.*rpm.install.excludedocs.*/rpm.install.excludedocs = no/g' /etc/zypp/zypp.conf
		fi

		# With recommended packages, something might try to pull in
		# parallel-printer-support which can't be installed in rootless containers.
		# Since we very much likely never need it, just lock it
		zypper al parallel-printer-support
		# Check if shell_pkg is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user.
		deps="
			bc
			man
			mtr
			pam
			zip
			zsh
			Mesa
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
			glibc
			gnupg
			rsync
			unzip
			words
			xauth
			ffmpeg
			libva2
			procps
			shadow
			iputils
			libdrm2
			libvpx2
			ncurses
			python3
			systemd
			wayland
			Mesa-dri
			hostname
			keyutils
			libopus0
			nss-mdns
			pinentry
			pipewire
			timezone
			xwayland
			diffutils
			findutils
			gstreamer
			libFLAC12
			libvte-2*
			man-pages
			pam-extra
			xdg-utils
			glibc-i18n
			libvulkan1
			mesa-libva
			util-linux
			libx264-164
			libx265-215
			wireplumber
			${shell_pkg}
			glibc-locale
			pipewire-alsa
			pipewire-jack
			xdg-user-dirs
			glibc-i18ndata
			bash-completion
			libvulkan_intel
			libwayland-egl1
			openssh-clients
			libvulkan_radeon
			glibc-locale-base
			intel-media-driver
			libva-intel-driver
			libwayland-client0
			libwayland-server0
			system-group-wheel
			util-linux-systemd
			xdg-desktop-portal
			glibc-locale-source
			pipewire-pulseaudio
			gstreamer-plugins-bad
			gstreamer-plugins-base
			gstreamer-plugins-good
			gstreamer-plugins-ugly
			gstreamer-plugins-libav
		"
		# zypper reserves exit codes >= 100 as informational: the transaction
		# committed, but something is worth noting - e.g. 106 (a repo was skipped) or
		# 107 (a package %post script failed, common in containers). Treat those as
		# non-fatal warnings. 105 (interrupted by a signal) shares the range but is a
		# genuine abort, so it stays fatal.
		# shellcheck disable=SC2086,SC2046
		zypper -n install -y $(zypper -n -q se --match-exact ${deps} | grep -e 'package$' | cut -d'|' -f2) || zypper_rc=${?}
		case "${zypper_rc:-0}" in
			0) ;;
			105) exit "${zypper_rc}" ;;
			1[0-9][0-9])
				printf "Warning: zypper exited %s (informational); packages installed.\n" "${zypper_rc}"
				;;
			*) exit "${zypper_rc}" ;;
		esac
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	# shellcheck disable=SC2154 # HOST_LOCALE, HOST_LOCALE_LANG, HOST_LOCALE_ENCODING assigned by otter-init before sourcing this file
	if ! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"; then
		LANG="${HOST_LOCALE}" localedef -i "${HOST_LOCALE_LANG}" -f "${HOST_LOCALE_ENCODING}" "${HOST_LOCALE}" || true
	fi

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		zypper install -f -y timezone
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		zypper install -y ${container_additional_packages}
	fi
}
