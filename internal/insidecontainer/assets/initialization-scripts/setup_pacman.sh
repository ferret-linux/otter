# setup_pacman_exceptions will set pre/post transaction hooks to avoid host's mounts, pacman only.
# Arguments:
#   None
# Expected global variables:
#   init: if this is an initful container
# Expected env variables:
#   None
# Outputs:
#   None
setup_pacman_exceptions()
{
	setup_pkg_manager_hooks

	# Workarounds for pacman. We need to exclude the paths by using a pre-hook to umount them and a
	# post-hook to remount them. Additionally we neutralize the systemd-post-hooks as they do not
	# work on a rootless container system.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ -d "/usr/share/libalpm/scripts" ] && [ "${init}" -eq 0 ]; then
		# in case we're not using an init image, neutralize systemd post installation hooks
		# so that we do not encounter problems along the way.
		# This will be removed if we're using --init.
		cat << EOF > /usr/share/libalpm/scripts/otter_post_hook.sh
#!/bin/sh
printf '#!/bin/sh\nexit 0\n' > /usr/share/libalpm/scripts/systemd-hook;
EOF
		chmod +x /usr/share/libalpm/scripts/otter_post_hook.sh

		# create hooks files for them
		find /usr/share/libalpm/hooks/*otter* -delete || :
		for hook in /etc/otter-pre-hook.sh /etc/otter-post-hook.sh /usr/share/libalpm/scripts/otter_post_hook.sh; do
			when="PostTransaction"
			[ -z "${hook##*pre*}" ] && when="PreTransaction"
			cat << EOF > "/usr/share/libalpm/hooks/$(basename "${hook}").hook"
[Trigger]
Operation = Install
Operation = Upgrade
Type = Package
Target = *
[Action]
Description = Otter hook ${hook}...
When = ${when}
Exec = ${hook}
EOF
		done
	fi

}

# setup_pacman will upgrade or setup all packages for pacman based systems.
# Arguments:
#   None
# Expected global variables:
#   upgrade: if we need to upgrade or not
#   container_additional_packages: additional packages to install during this phase
# Expected env variables:
#   None
# Outputs:
#   None
setup_pacman()
{
	# Update the package repository cache exactly once before installing packages.
	pacman -S -y -y

	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		pacman -S -u --noconfirm
		exit
	fi
	# In archlinux official images, pacman is configured to ignore locale and docs
	# This however, results in a rather poor out-of-the-box experience
	# So, let's enable them.
	sed -i "s|NoExtract.*||g" /etc/pacman.conf
	sed -i "s|NoProgressBar.*||g" /etc/pacman.conf
	sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf
    sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

	pacman -S -u --noconfirm
	# Check if shell_pkg is available in distro's repo. If not we
	# fall back to bash, and we set the SHELL variable to bash so
	# that it is set up correctly for the user.
	if [ ! -f /usr/lib/otter/container.official ]; then
		if ! pacman -S --needed --noconfirm "${shell_pkg}"; then
			shell_pkg="bash"
		fi
	fi
	distro_id="$(get_distro_id)"
	case "${distro_id}" in
		arch)
			distro_deps="
				base
				base-devel
				ffmpeg
				fakeroot
				fish
				git
				gst-plugin-pipewire
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				libva
				pipewire
				pipewire-alsa
				pipewire-jack
				pipewire-pulse
				systemd
				tar
				vpl-gpu-rt
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xorg-xwayland
				zsh
			"
			;;
		artix)
			distro_deps="
				base
				base-devel
				ffmpeg
				fakeroot
				fish
				git
				gst-plugin-pipewire
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				libva
				openrc
				pipewire
				pipewire-alsa
				pipewire-jack
				pipewire-pulse
				tar
				vpl-gpu-rt
				wayland
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xorg-xwayland
				zsh
			"
			;;
		blackarch)
			distro_deps="
				aircrack-ng
				base
				base-devel
				bettercap
				binwalk
				bully
				checksec
				crunch
				dnsenum
				dsniff
				ettercap
				exiftool
				ffmpeg
				ffuf
				fakeroot
				fish
				foremost
				git
				gobuster
				gst-plugin-pipewire
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				hashcat
				hcxtools
				hydra
				john
				libva
				masscan
				medusa
				mitmproxy
				nikto
				nmap
				pipewire
				pipewire-alsa
				pipewire-jack
				pipewire-pulse
				radare2
				reaver
				sqlmap
				systemd
				tar
				theharvester
				tshark
				vpl-gpu-rt
				whatweb
				wifite
				wpscan
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				xorg-xwayland
				zsh
			"
			;;
		cachyos)
			distro_deps="
				base
				base-devel
				fakeroot
				ffmpeg
				fish
				git
				gst-plugins-bad
				gst-plugins-base
				gst-plugins-good
				gst-plugins-ugly
				paru
				pipewire
				pipewire-jack
				pipewire-pulse
				proton
				proton-cachyos-slr
				protonplus
				protontricks
				systemd
				tar
				umu-launcher
				vpl-gpu-rt
				vulkan-driver
				wine
				wine-cachyos-opt
				wine-gecko
				wine-mono
				xdg-desktop-portal
				xdg-user-dirs
				xdg-utils
				zsh
			"
			;;
		*)
			printf "Error: unsupported pacman-based distro '%s'.\n" "${distro_id}"
			printf "Error: could not set up base dependencies.\n"
			exit 127
			;;
	esac
	deps="${distro_deps}
		${shell_pkg}
		bash-completion
		bc
		curl
		diffutils
		findutils
		glibc
		gnupg
		iputils
		inetutils
		keyutils
		less
		lsof
		man-db
		man-pages
		mlocate
		mtr
		ncurses
		nss-mdns
		openssh
		pigz
		pinentry
		procps-ng
		python
		rsync
		shadow
		sudo
		tcpdump
		time
		traceroute
		tree
		tzdata
		unzip
		util-linux
		util-linux-libs
		vte-common
		wget
		words
		xorg-xauth
		zip
		mesa
		vulkan-intel
		vulkan-radeon
	"
	# shellcheck disable=SC2086,2046
	pacman -S --needed --noconfirm $(pacman -Ssq | grep -E "^($(echo ${deps} | tr ' ' '|'))$")

	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ ! -e "/usr/share/i18n/locales${HOST_LOCALE}" ]; then
		pacman -S --noconfirm glibc glibc-locales
	fi

	# Ensure we have tzdata installed and populated, sometimes container
	# images blank the zoneinfo directory, so we reinstall the package to
	# ensure population
	if [ ! -e /usr/share/zoneinfo/UTC ]; then
		pacman -S --noconfirm tzdata
	fi

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
	if ! locale -a | grep -qi en_us.utf8 || ! locale -a | grep -qi "$(echo "${HOST_LOCALE}" | tr -d '-')"; then
		sed -i "s|#.*en_US.UTF-8|en_US.UTF-8|g" /etc/locale.gen
		sed -i "s|#.*${HOST_LOCALE}|${HOST_LOCALE}|g" /etc/locale.gen
		locale-gen -a
	fi

	# Install additional packages passed at otter-create time
	if [ -n "${container_additional_packages}" ]; then
		# shellcheck disable=SC2086
		pacman -S --needed --noconfirm ${container_additional_packages}
	fi
}
