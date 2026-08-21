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
	deps="
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
