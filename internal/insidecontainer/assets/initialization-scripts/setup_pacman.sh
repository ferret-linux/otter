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
	# If we need to upgrade, do it and exit, no further action required.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	if [ "${upgrade}" -ne 0 ]; then
		pacman -S -y -y
		pacman -S -u --noconfirm
		exit
	fi
	if [ ! -f /usr/lib/otter/container.official ]; then
		# Update the package repository cache exactly once before installing packages.
		pacman -S -y -y

		# In archlinux official images, pacman is configured to ignore locale and docs
		# This however, results in a rather poor out-of-the-box experience
		# So, let's enable them.
		sed -i "s|NoExtract.*||g" /etc/pacman.conf
		sed -i "s|NoProgressBar.*||g" /etc/pacman.conf
		sed -i '/^\s*#\?\s*ILoveCandy/d; /^\s*#\?\s*Color/d' /etc/pacman.conf
	    sed -i '0,/^\[options\]/s/^\[options\]/\[options\]\nColor\nILoveCandy/' /etc/pacman.conf

		pacman -S -u --noconfirm
		# Check if shell_bin is available in distro's repo. If not we
		# fall back to bash, and we set the SHELL variable to bash so
		# that it is set up correctly for the user. nu's pacman package
		# is named nushell, not nu.
		case "${shell_bin}" in
			nu) shell_pkg="nushell" ;;
			*) shell_pkg="${shell_bin}" ;;
		esac
		if ! pacman -S --needed --noconfirm "${shell_pkg}"; then
			shell_bin="bash"
		fi
		distro_id="$(get_distro_id)"
		case "${distro_id}" in
			arch)
				distro_deps="
					systemd
				"
				;;
			artix)
				distro_deps="
					openrc
					wayland
				"
				;;
			blackarch)
				distro_deps="
					ffuf
					john
					nmap
					bully
					hydra
					nikto
					crunch
					dsniff
					medusa
					reaver
					sqlmap
					tshark
					wifite
					wpscan
					binwalk
					dnsenum
					hashcat
					masscan
					radare2
					systemd
					whatweb
					checksec
					ettercap
					exiftool
					foremost
					gobuster
					hcxtools
					bettercap
					mitmproxy
					aircrack-ng
					theharvester
				"
				;;
			cachyos)
				distro_deps="
					paru
					wine
					proton
					systemd
					wine-mono
					protonplus
					wine-gecko
					protontricks
					umu-launcher
					vulkan-driver
					wine-cachyos-opt
					proton-cachyos-slr
				"
				;;
			*)
				printf "Error: unsupported pacman-based distro '%s'.\n" "${distro_id}"
				printf "Error: could not set up base dependencies.\n"
				exit 127
				;;
		esac
		deps="${distro_deps}
			bc
			git
			mtr
			tar
			zip
			zsh
			base
			curl
			fish
			less
			lsof
			mesa
			pigz
			sudo
			time
			tree
			wget
			glibc
			gnupg
			libva
			rsync
			unzip
			words
			ffmpeg
			man-db
			python
			shadow
			tzdata
			iputils
			mlocate
			ncurses
			openssh
			tcpdump
			fakeroot
			keyutils
			nss-mdns
			pinentry
			pipewire
			diffutils
			findutils
			inetutils
			man-pages
			procps-ng
			xdg-utils
			base-devel
			traceroute
			util-linux
			vpl-gpu-rt
			vte-common
			xorg-xauth
			${shell_pkg}
			vulkan-intel
			pipewire-alsa
			pipewire-jack
			vulkan-radeon
			xdg-user-dirs
			xorg-xwayland
			pipewire-pulse
			bash-completion
			gst-plugins-bad
			util-linux-libs
			gst-plugins-base
			gst-plugins-good
			gst-plugins-ugly
			xdg-desktop-portal
			gst-plugin-pipewire
		"
		# shellcheck disable=SC2086,2046
		pacman -S --needed --noconfirm $(pacman -Ssq | grep -E "^($(echo ${deps} | tr ' ' '|'))$")
	fi

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
