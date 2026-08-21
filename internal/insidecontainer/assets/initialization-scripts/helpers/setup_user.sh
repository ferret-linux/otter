# setup_user ensures the container's primary user and its group exist with
# the requested uid/gid/home/shell, reconciling a pre-existing (e.g.
# podman/docker-created) user if needed, and sets up passwordless/first-shell
# password handling for root and the user.
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user
#   container_user_uid: the container's primary user's uid
#   container_user_gid: the container's primary user's gid
#   container_user_home: the container's primary user's home directory
#   rootful: whether the container is rootful
# Expected env variables:
#   SHELL
# Outputs:
#   None
setup_user()
{
###############################################################################
# If not existing, ensure we have a group for our user.
# shellcheck disable=SC2154 # assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
if ! grep -q "^${container_user_name}:" /etc/group; then
	printf "otter: Setting up user groups...\n"

	# shellcheck disable=SC2154 # assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
	if ! groupadd --force --gid "${container_user_gid}" "${container_user_name}"; then
		# It may occur that we have users with unsupported user name (eg. on LDAP or AD)
		# So let's try and force the group creation this way.
		printf "%s:x:%s:\n" "${container_user_name}" "${container_user_gid}" >> /etc/group
	fi
fi
###############################################################################

###############################################################################

# Setup kerberos integration with the host
if [ -d "/run/host/var/kerberos" ] &&
	[ -d "/etc/krb5.conf.d" ] &&
	[ ! -e "/etc/krb5.conf.d/kcm_default_ccache" ]; then

	printf "otter: Setting up kerberos integration...\n"

	cp /usr/lib/otter/scripts/initialization-scripts/configs/krb5-kcm-default-ccache.conf /etc/krb5.conf.d/kcm_default_ccache
fi

printf "otter: Setting up user's group list...\n"
# If we have sudo/wheel groups, let's add the user to them.
# and ensure that user's in those groups can effectively sudo
additional_groups=""
if grep -q "^sudo" /etc/group; then
	additional_groups="sudo"
elif grep -q "^wheel" /etc/group; then
	additional_groups="wheel"
elif grep -q "^root" /etc/group; then
	additional_groups="root"
fi

# Let's add our user to the container. if the user already exists, enforce properties.
#
# In case of AD or LDAP usernames, it is possible we will have a backslach in the name.
# In that case grep would fail, so we replace the backslash with a point to make the regex work.
# shellcheck disable=SC1003,SC2154 # SC2154: container_user_uid assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
if ! grep -q "^$(printf '%s' "${container_user_name}" | tr '\\' '.'):" /etc/passwd &&
	! getent passwd "${container_user_uid}"; then
	printf "otter: Adding user...\n"
	# shellcheck disable=SC2154 # assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
	if ! useradd \
		--home-dir "${container_user_home}" \
		--no-create-home \
		--groups "${additional_groups}" \
		--shell "${SHELL:-"/bin/bash"}" \
		--uid "${container_user_uid}" \
		--gid "${container_user_gid}" \
		"${container_user_name}"; then

		printf "Warning: There was a problem setting up the user with usermod, trying manual addition\n"

		printf "%s:x:%s:%s:%s:%s:%s\n" \
			"${container_user_name}" "${container_user_uid}" \
			"${container_user_gid}" "${container_user_name}" \
			"${container_user_home}" "${SHELL:-"/bin/bash"}" >> /etc/passwd
		printf "%s::1::::::" "${container_user_name}" >> /etc/shadow

		# Also add user to any additional groups when useradd failed
		for group in ${additional_groups}; do
			if ! grep -q "^${group}.*${container_user_name}" /etc/group; then
				group_line="$(grep "^${group}.*" /etc/group)"
				if grep -q "^${group}.*:$" /etc/group; then
					sed -i "s|${group_line}|${group_line}${container_user_name}|g" /etc/group
				else
					sed -i "s|${group_line}|${group_line},${container_user_name}|g" /etc/group
				fi
			fi
		done
	fi
# Ensure we're not using the specified SHELL. Run it only once, so that future
# user's preferences are not overwritten at each start.
elif [ ! -e /etc/passwd.done ]; then
	# This situation is presented when podman or docker already creates the user
	# for us inside container. We should modify the user's prepopulated shadowfile
	# entry though as per user's active preferences.

	# Get current user attributes using container_user_uid as the reference
	# (script runs as root, so we must look up the target user by UID)
	current_user_entry=$(getent passwd "${container_user_uid}")
	current_user_name=$(printf '%s' "${current_user_entry}" | cut -d: -f1)
	current_shell=$(printf '%s' "${current_user_entry}" | cut -d: -f7)
	current_gid=$(printf '%s' "${current_user_entry}" | cut -d: -f4)
	current_groups=$(id -nG "${current_user_name}" 2> /dev/null)

	# Modify username if needed
	if [ "${current_user_name}" != "${container_user_name}" ]; then
		printf "otter: Setting up existing user - username...\n"
		if ! usermod --login "${container_user_name}" "${current_user_name}"; then
			printf "Warning: usermod --login failed, trying manual modification\n"
			sed -i "s|^${current_user_name}:|${container_user_name}:|g" /etc/passwd
			if ! getent passwd "${container_user_name}" > /dev/null 2>&1; then
				printf "Error: Failed to modify user login name\n" >&2
				exit 1
			fi
		fi
		# Update current_user_name for subsequent commands
		current_user_name="${container_user_name}"
	fi

	# Modify shell if needed
	if [ "${current_shell}" != "${SHELL:-"/bin/bash"}" ]; then
		printf "otter: Setting up existing user - shell...\n"
		if ! usermod --shell "${SHELL:-"/bin/bash"}" "${current_user_name}"; then
			printf "Warning: usermod --shell failed, trying manual modification\n"
			# sed to update shell field (7th field) in /etc/passwd
			sed -i "s|^\(${current_user_name}:[^:]*:[^:]*:[^:]*:[^:]*:[^:]*:\).*|\1${SHELL:-"/bin/bash"}|g" /etc/passwd
		fi
	fi

	# Modify GID if needed
	if [ "${current_gid}" != "${container_user_gid}" ]; then
		printf "otter: Setting up existing user - GID...\n"
		if ! usermod --gid "${container_user_gid}" "${current_user_name}"; then
			printf "Warning: usermod --gid failed, trying manual modification\n"
			# sed to update gid field (4th field) in /etc/passwd
			sed -i "s|^\(${current_user_name}:[^:]*:[^:]*:\)[^:]*|\1${container_user_gid}|g" /etc/passwd
		fi
	fi

	# Modify groups if needed (check if user is missing from any additional group)
	groups_need_modification=0
	for group in ${additional_groups}; do
		if ! printf '%s' " ${current_groups} " | grep -q " ${group} "; then
			groups_need_modification=1
			break
		fi
	done
	if [ "${groups_need_modification}" -eq 1 ]; then
		printf "otter: Setting up existing user - groups...\n"
		# Workaround: usermod --groups fails if an /etc/group file does not end with
		# a newline, so we preemptively add one just in case.
		printf '\n' >> /etc/group
		if ! usermod --append --groups "${additional_groups}" "${current_user_name}"; then
			printf "Warning: usermod --groups failed, trying manual modification\n"
			for group in ${additional_groups}; do
				if grep -q "^${group}:" /etc/group && ! grep -q "^${group}.*${current_user_name}.*" /etc/group; then
					group_line="$(grep "^${group}:" /etc/group)"
					if grep -q "^${group}:.*:$" /etc/group; then
						sed -i "s|${group_line}|${group_line}${current_user_name}|g" /etc/group
					else
						sed -i "s|${group_line}|${group_line},${current_user_name}|g" /etc/group
					fi
				fi
			done
		fi
	fi

	# Modify UID if needed
	current_uid=$(getent passwd "${current_user_name}" | cut -d: -f3)
	if [ "${current_uid}" != "${container_user_uid}" ]; then
		printf "otter: UID...\n"
		if ! usermod --uid "${container_user_uid}" "${current_user_name}"; then
			printf "Warning: usermod --uid failed, trying manual modification\n"
			sed -i "s|^\(${current_user_name}:[^:]*:\)[^:]*|\1${container_user_uid}|g" /etc/passwd
		fi
	fi
fi

# Ensure we have our home correctly set, in case of cloned containers or whatnot
if [ "$(getent passwd "${container_user_name}" | cut -d: -f6)" != "${container_user_home}" ]; then
	printf "otter: Setting up user home...\n"

	if ! usermod -d "${container_user_home}" "${container_user_name}"; then
		sed -i "s|^${container_user_name}.*|${container_user_name}:x:${container_user_uid}:${container_user_gid}::${container_user_home}:${SHELL:-"/bin/bash"}|g" /etc/passwd
	fi
fi

# If we're rootless, delete password for root and user
if [ ! -e /etc/passwd.done ]; then
	printf "otter: Ensuring user's access...\n"

	temporary_password="$(md5sum < /proc/sys/kernel/random/uuid | cut -d' ' -f1)"
	# We generate a random password to initialize the entry for the user.
	chpasswd_failed=0
	printf "%s:%s" "${container_user_name}" "${temporary_password}" | chpasswd || chpasswd_failed=1
	# Then we remove the password for current user
	if ! passwd -d "${container_user_name}"; then
		# Fallback to chpasswd for older systems without passwd -d
		printf "%s:" "${container_user_name}" | chpasswd || chpasswd_failed=1
	fi

	if [ "${chpasswd_failed}" -eq 1 ]; then
		printf "Warning: There was a problem setting up the user, trying manual addition\n"
		if grep -q "${container_user_name}" /etc/shadow; then
			sed -i "s|^${container_user_name}.*|${container_user_name}::::::::|g" /etc/shadow
		else
			echo "${container_user_name}::::::::" >> /etc/shadow
		fi
	fi

	# shellcheck disable=SC2154 # assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
	if [ "${rootful}" -eq 0 ]; then
		# We're rootless so we don't care about account password, so we remove it
		passwd_cmd=passwd
		if passwd --help 2>&1 | grep -q -- --stdin; then
			passwd_cmd="passwd --stdin"
		fi
		printf "%s\n%s\n" "${temporary_password}" "${temporary_password}" | ${passwd_cmd} root
		if ! passwd -d "root"; then
			# Fallback to chpasswd for older systems without passwd -d
			printf "%s:" "root" | chpasswd
		fi
	else
		# We're rootful, so we don't want passwordless accounts, so we lock them
		# down by default.

		# lock out root user
		if ! usermod -L root; then
			sed -i 's|^root.*|root:!:1::::::|g' /etc/shadow
		fi
	fi
fi

# If we are in a rootful container, let's setup a first-shell password setup
# so that sudo, and su has a password
#
# else we fallback to the usual setup with passwordless sudo/su user. This is
# likely because we're in a rootless setup, so privilege escalation is not a concern.
if [ "${rootful}" -eq 1 ] &&
	{
		[ "$(grep "${container_user_name}" /etc/shadow | cut -d':' -f2)" = '!!' ] ||
			[ "$(grep "${container_user_name}" /etc/shadow | cut -d':' -f2)" = "" ]
	}; then

	# force setup of user's password on first shell
	if [ ! -e /var/tmp ]; then
		mkdir -p /var/tmp
		chmod 0777 /var/tmp
	fi
	touch /var/tmp/."${container_user_name}".passwd.initialize
	chown "${container_user_name}:${container_user_gid}" /var/tmp/."${container_user_name}".passwd.initialize
fi

# Now we're done
touch /etc/passwd.done

# Ensure shadow files are readable by root without relying on CAP_DAC_OVERRIDE,
# which may not be effective on all container storage drivers (e.g. fuse-overlayfs
# in rootless mode, or VMs like Docker Desktop / Colima on macOS).
# Fedora/Arch ship these as mode 000, expecting the capability to bypass DAC.
chmod 0400 /etc/shadow 2> /dev/null || :
chmod 0400 /etc/gshadow 2> /dev/null || :
###############################################################################
}