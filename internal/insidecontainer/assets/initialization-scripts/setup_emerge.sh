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
	if [ "${upgrade}" -ne 0 ]; then
		emerge --sync
		exit
	fi
	# Check if shell_pkg is available in distro's repo. If not we
	# fall back to bash, and we set the SHELL variable to bash so
	# that it is set up correctly for the user.
	getuto 2>/dev/null || :
	if [ ! -f /usr/lib/otter/container.official ]; then
		if ! emerge --ask=n --autounmask-continue --noreplace --quiet-build --getbinpkg "${shell_pkg}"; then
			shell_pkg="bash"
		fi
	fi
	deps="
		app-shells/${shell_pkg}
		app-crypt/gnupg
		app-shells/bash-completion
		sys-apps/diffutils
		sys-apps/findutils
		sys-apps/less
		sys-libs/ncurses
		net-misc/curl
		app-crypt/pinentry
		sys-process/procps
		sys-apps/shadow
		app-admin/sudo
		sys-devel/bc
		sys-process/lsof
		sys-apps/util-linux
		net-misc/wget
		app-arch/pigz
		app-text/tree
		x11-apps/xauth
		dev-lang/python
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

	# In case the locale is not available, install it
	# will ensure we don't fallback to C.UTF-8
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
