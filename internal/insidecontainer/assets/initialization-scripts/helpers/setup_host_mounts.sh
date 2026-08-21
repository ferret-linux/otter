# setup_host_mounts bind mounts read-only and read-write host paths into the
# container (with fallback hard-copy for read-only files), and links host
# sockets (e.g. podman/docker/libvirt) into the container.
# Arguments:
#   None
# Expected global variables:
#   HOST_MOUNTS_RO: list of read-only host mountpoints
#   HOST_MOUNTS: list of read-write host mountpoints
#   container_user_name: the container's primary user
#   OTTER_HOST_HOME: set when a custom home is in use (optional)
# Expected env variables:
#   None
# Outputs:
#   None
setup_host_mounts()
{
###############################################################################
printf "otter: Setting up read-only mounts...\n"

# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
for host_mount_ro in ${HOST_MOUNTS_RO}; do
	# Mounting read-only in a user namespace will trigger a check to see if certain
	# "locked" flags (line noexec,nodev,nosuid) are changed. This ensures we explicitly reuse those flags.
	locked_flags="$(get_locked_mount_flags /run/host"${host_mount_ro}")"
	if ! mount_bind /run/host"${host_mount_ro}" "${host_mount_ro}" ro"${locked_flags:+,${locked_flags}}"; then
		printf "Warning: %s integration with the host failed, runtime sync for %s disabled.\n" "${host_mount_ro}" "${host_mount_ro}"
		# Fallback options for files, we do a hard copy of it
		if [ -f /run/host"${host_mount_ro}" ]; then
			if ! (rm -f "${host_mount_ro}" && cp -f /run/host"${host_mount_ro}" "${host_mount_ro}"); then
				printf "Warning: Hard copy failed. Error: %s\n" "$(cp -f /run/host"${host_mount_ro}" "${host_mount_ro}" 2>&1)"
			fi
		fi
	fi
done
###############################################################################

###############################################################################
printf "otter: Setting up read-write mounts...\n"

# On some ostree systems, home is in /var/home, but most of the software expects
# it to be in /home. In the hosts systems this is fixed by using a symlink.
# Do something similar here with a bind mount. Skip this entirely when a custom
# home is in use (OTTER_HOST_HOME set), since a custom home should stay isolated
# from the real host home.
# shellcheck disable=SC2154 # assigned by parse_args.sh (sourced+called by otter-init) before sourcing this file
if [ -z "${OTTER_HOST_HOME-}" ] && [ -e "/var/home/${container_user_name}" ]; then
	if ! mount_bind "/run/host/var/home/${container_user_name}" "/home/${container_user_name}"; then
		printf "Warning: Cannot bind mount %s to /run/host%s\n" "/var/home" "/home"
	fi
fi

# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
for host_mount in ${HOST_MOUNTS}; do
	if ! mount_bind /run/host"${host_mount}" "${host_mount}"; then
		printf "Warning: Cannot bind mount %s to /run/host%s\n" "${host_mount}" "${host_mount}"
	fi
done
###############################################################################

###############################################################################
printf "otter: Setting up host's sockets integration...\n"
# Find all the user's socket and mount them inside the container
# this will allow for continuity of functionality between host and container
#
# for example using `podman --remote` to control the host's podman from inside
# the container or accessing docker and libvirt sockets.
host_sockets="$(find /run/host/run \
	-xdev \
	-path /run/host/run/media -prune -o \
	-path /run/host/run/timeshift -prune -o \
	-name 'user' -prune -o \
	-name 'bees' -prune -o \
	-name 'nscd' -prune -o \
	-name 'schroot' -prune -o \
	-name 'system_bus_socket' -prune -o \
	-name 'io.systemd.Multiplexer' -prune -o \
	-name 'io.systemd.DropIn' -prune -o \
	-name 'io.systemd.NameServiceSwitch' -prune -o \
	-type s -print \
	2> /dev/null || :)"

# we're excluding system dbus socket, nscd socket and systemd-userdbd sockets here. Including them will
# create many problems with package managers thinking they have access to
# system dbus, user auth cache misused or query wrong user information.
for host_socket in ${host_sockets}; do
	container_socket="${host_socket#/run/host}"
	# Check if the socket already exists or the symlink already exists
	if [ ! -S "${container_socket}" ] && [ ! -L "${container_socket}" ]; then
		# link it.
		rm -f "${container_socket}"
		mkdir -p "$(dirname "${container_socket}")"
		if ! ln -s "${host_socket}" "${container_socket}"; then
			printf "Warning: Cannot link socket %s to %s\n" "${host_socket}" "${container_socket}"
		fi
	fi
done
###############################################################################
}
