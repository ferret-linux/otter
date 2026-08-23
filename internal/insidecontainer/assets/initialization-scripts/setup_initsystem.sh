# setup_initsystem fires up the container's init system (systemd or generic
# sysvinit-style /sbin/init), masking host-conflicting units and setting up
# console/user integration first. Detection happens here; each supported init
# system does its own setup/launch in a dedicated setup_init_<name> function.
# Arguments:
#   None
# Expected global variables:
#   HOST_MOUNTS_RO_INIT: list of readonly init-time mountpoints to unmount
#   id: container id, used to detect host-owned mounts
#   container_user_name: the container's primary user, for user@.service/lingering
# Expected env variables:
#   None
# Outputs:
#   None
setup_initsystem()
{
	###############################################################################
	# If we're here, the init support has been enabled.
	printf "otter: Setting up init system...\n"

	# some of this directories are needed by
	# the init system. If they're mounts, there might
	# be problems. Let's unmount them.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	for host_mount in ${HOST_MOUNTS_RO_INIT}; do
		if findmnt "${host_mount}" > /dev/null; then umount "${host_mount}"; fi
	done

	# Restore the symlink to host's timezone
	if [ -f /run/host/etc/localtime ]; then
		rm -f /etc/localtime
		ln -sf /run/host/etc/localtime /etc/localtime
	fi

	# Remove /dev/console when using init systems, this will confuse host system if
	# we use rootful containers
	# Instantiate a new pty to mount over /dev/console
	# this way we will have init output right of the logs
	[ -e /dev/console ] || touch /dev/console
	rm -f /var/console
	mkfifo /var/console
	script -c "cat /var/console" /dev/null &

	# Ensure the pty is created
	sleep 0.5

	# Mount the created pty over /dev/console in order to have systemd logs
	# right into container logs
	if ! mount --bind /dev/pts/0 /dev/console; then
		# Fallback to older behaviour or fake plaintext file in case it fails
		# this ensures rootful + initful boxes do not interfere with host's /dev/console
		rm -f /var/console
		touch /var/console
		mount --bind /var/console /dev/console
	fi

	if [ -e /etc/inittab ]; then
		# Cleanup openrc to not interfere with the host
		sed -i 's/^\(tty\d\:\:\)/#\1/g' /etc/inittab
	fi

	if [ -e /etc/rc.conf ]; then
		sed -i \
			-e 's/#rc_env_allow=".*"/rc_env_allow="\*"/g' \
			-e 's/#rc_crashed_stop=.*/rc_crashed_stop=NO/g' \
			-e 's/#rc_crashed_start=.*/rc_crashed_start=YES/g' \
			-e 's/#rc_provide=".*"/rc_provide="loopback net"/g' \
			/etc/rc.conf
	fi

	if [ -e /etc/init.d ]; then
		rm -f /etc/init.d/hwdrivers \
			/etc/init.d/hwclock \
			/etc/init.d/hwdrivers \
			/etc/init.d/modules \
			/etc/init.d/modules-load \
			/etc/init.d/modloop
	fi

	# Now we can launch init
	printf "otter: Firing up init system...\n"

	if [ -e /usr/lib/systemd/systemd ] || [ -e /lib/systemd/systemd ]; then
		setup_init_systemd
	elif command -v runlevel > /dev/null 2>&1; then
		setup_init_sysvinit
	elif [ -e /sbin/init ]; then
		setup_init_generic
	else
		printf "Error: could not set up init system, no init found! Consider using an image that ships with an init system, or add it with \"--additional-packages\" during creation.!\n"
		exit 1
	fi
}

# write_user_integration_script generates the shared /usr/local/bin/user-integration
# script that mirrors host X11/Wayland/Pipewire/D-Bus/keyring sockets into the
# container user's runtime dir. The script itself is init-agnostic; only how/when
# each init triggers it differs (systemd via a oneshot unit, sysvinit via an LSB
# init script).
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
write_user_integration_script()
{
	cat << EOF > /usr/local/bin/user-integration
#!/bin/sh
sleep 1
ln -sf /run/host/run/user/\$(id -ru)/wayland-* /run/user/\$(id -ru)/
ln -sf /run/host/run/user/\$(id -ru)/pipewire-* /run/user/\$(id -ru)/
find /run/host/run/user/\$(id -ru)/ -maxdepth 1 -type f -exec sh -c 'grep -qlE COOKIE \$0 && ln -sf \$0 /run/user/\$(id -ru)/\$(basename \$0)' {} \;
mkdir -p /run/user/\$(id -ru)/app && ln -sf /run/host/run/user/\$(id -ru)/app/* /run/user/\$(id -ru)/app/
mkdir -p /run/user/\$(id -ru)/at-spi && ln -sf /run/host/run/user/\$(id -ru)/at-spi/* /run/user/\$(id -ru)/at-spi/
mkdir -p /run/user/\$(id -ru)/dbus-1 && ln -sf /run/host/run/user/\$(id -ru)/dbus-1/* /run/user/\$(id -ru)/dbus-1/
mkdir -p /run/user/\$(id -ru)/dconf && ln -sf /run/host/run/user/\$(id -ru)/dconf/* /run/user/\$(id -ru)/dconf/
mkdir -p /run/user/\$(id -ru)/gnupg && ln -sf /run/host/run/user/\$(id -ru)/gnupg/* /run/user/\$(id -ru)/gnupg/
mkdir -p /run/user/\$(id -ru)/keyring && ln -sf /run/host/run/user/\$(id -ru)/keyring/* /run/user/\$(id -ru)/keyring/
mkdir -p /run/user/\$(id -ru)/p11-kit && ln -sf /run/host/run/user/\$(id -ru)/p11-kit/* /run/user/\$(id -ru)/p11-kit/
mkdir -p /run/user/\$(id -ru)/pulse && ln -sf /run/host/run/user/\$(id -ru)/pulse/* /run/user/\$(id -ru)/pulse/
find /run/user/\$(id -ru) -maxdepth 2 -xtype l -delete
EOF

	chmod +x /usr/local/bin/user-integration
}

# setup_init_systemd masks host-conflicting systemd units, wires up minimal
# user-session integration (Wayland/Pipewire/D-Bus/keyring), then launches
# systemd and waits for it to report ready before finishing container setup.
# Arguments:
#   None
# Expected global variables:
#   id: container id, used to detect host-owned mounts
#   container_user_name: the container's primary user, for user@.service/lingering
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_systemd()
{
	# Remove symlinks
	rm -f /run/systemd/coredump
	rm -f /run/systemd/io.system.ManagedOOM
	rm -f /run/systemd/notify
	rm -f /run/systemd/private

	if command -v systemctl 2> /dev/null; then
		# Cleanup Systemd to not interfere with the host
		UNIT_TARGETS="
			/usr/lib/systemd/system/*.mount
			/usr/lib/systemd/system/console-getty.service
			/usr/lib/systemd/system/getty@.service
			/usr/lib/systemd/system/systemd-machine-id-commit.service
			/usr/lib/systemd/system/systemd-binfmt.service
			/usr/lib/systemd/system/systemd-tmpfiles*
			/usr/lib/systemd/system/systemd-udevd.service
			/usr/lib/systemd/system/systemd-udev-trigger.service
			/usr/lib/systemd/system/systemd-update-utmp*
			/usr/lib/systemd/user/pipewire*
			/usr/lib/systemd/user/wireplumber*
			/usr/lib/systemd/system/suspend.target
			/usr/lib/systemd/system/hibernate.target
			/usr/lib/systemd/system/hybrid-sleep.target
			/usr/lib/systemd/system/systemd-remount-fs.service
		"

		# in case /etc/resolv.conf is a mount, we need to mask resolved
		# in this case we're using network=host and systemd-resolved won't
		# be able to bind to localhost:53
		mount_source="$(findmnt -no SOURCE /etc/resolv.conf)" || :
		# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
		if [ -n "${mount_source}" ] && ! echo "${mount_source}" | grep -q "${id}"; then
			UNIT_TARGETS="${UNIT_TARGETS}
				/usr/lib/systemd/system/systemd-resolved.service
			"
		fi

		# shellcheck disable=SC2086,SC2044
		for unit in $(find ${UNIT_TARGETS} 2> /dev/null); do
			systemctl mask "$(basename "${unit}")" || :
		done
	fi

	# Let's do a minimal user-integration for the user when using system
	# as the user@.service will trigger the user-runtime-dir@.service which will
	# undo all the integration we did at the start of the script
	#
	# This will ensure the basic integration for x11/wayland/pipewire/keyring
	if [ -e /usr/lib/systemd/system/user@.service ]; then
		write_user_integration_script

		cat << EOF > /usr/lib/systemd/system/user-integration@.service
[Unit]
Description=User runtime integration for UID %i
After=user@%i.service
Requires=user-runtime-dir@%i.service

[Service]
User=%i
Type=oneshot
ExecStart=/usr/local/bin/user-integration

Slice=user-%i.slice
EOF
	fi

	# Start user Systemd unit, this will attempt until Systemd is ready
	# shellcheck disable=SC2154 # container_user_name assigned by otter-init before sourcing this file
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    systemctl is-system-running | grep -E 'running|degraded' && break; \
		echo 'waiting for systemd to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	systemctl start user@${container_user_name}.service && \
	systemctl start user-integration@${container_user_name}.service && \
	loginctl enable-linger ${container_user_name} || : && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	[ -e /usr/lib/systemd/systemd ] && exec /usr/lib/systemd/systemd --system --log-target=console --unit=multi-user.target
	[ -e /lib/systemd/systemd ] && exec /lib/systemd/systemd --system --log-target=console --unit=multi-user.target
}

# setup_init_sysvinit disables host-conflicting getty respawns, wires up the
# host user-session integration via an LSB init script (sysvinit has no
# user@.service equivalent to hook into), waits for a valid runlevel, then
# launches sysvinit. Detected via the "runlevel" command (shipped by
# sysvinit-utils), not by distro, since /sbin/init and /etc/inittab alone
# can't reliably distinguish sysvinit from other inits (e.g. busybox-init/
# OpenRC also ship an /sbin/init and inittab-style file).
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the user-integration
#     LSB script
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_sysvinit()
{
	# Classic sysvinit inittab uses "N:RUNLEVELS:ACTION:PROCESS" lines, e.g.
	# "1:2345:respawn:/sbin/getty 38400 tty1" — different syntax from the
	# BusyBox/OpenRC "ttyN::respawn:..." lines already handled above. Comment
	# these out too, so sysvinit doesn't spawn gettys that conflict with the host.
	if [ -e /etc/inittab ]; then
		sed -i 's/^\([^:]*:[^:]*:respawn:.*getty.*\)/#\1/' /etc/inittab
	fi

	write_user_integration_script

	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	cat << EOF > /etc/init.d/otter-user-integration
#!/bin/sh
### BEGIN INIT INFO
# Provides:          otter-user-integration
# Required-Start:    \$local_fs \$remote_fs
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:
# Short-Description: Otter host user-session integration
### END INIT INFO

case "\$1" in
	start)
		su - "${container_user_name}" -c /usr/local/bin/user-integration
		;;
	stop | restart | reload | force-reload | status)
		;;
	*)
		echo "Usage: \$0 start" >&2
		exit 1
		;;
esac

exit 0
EOF

	chmod +x /etc/init.d/otter-user-integration

	if command -v update-rc.d > /dev/null 2>&1; then
		update-rc.d otter-user-integration defaults > /dev/null 2>&1 || :
	fi

	# Wait for sysvinit to report a real runlevel before marking the
	# container ready. "runlevel" prints "<previous> <current>", e.g.
	# "N 2"; it prints "unknown" for both fields before the switch happens.
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    current_runlevel=\$(runlevel | awk '{print \$2}'); \
	    [ \"\${current_runlevel}\" != \"unknown\" ] && [ -n \"\${current_runlevel}\" ] && break; \
		echo 'waiting for sysvinit to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	exec /sbin/init
}

# setup_init_generic is the fallback path used for any non-systemd init that
# ships a standard /sbin/init entrypoint (e.g. an openrc alpine image today).
# It performs no init-specific readiness wait or user-session integration.
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_generic()
{
	touch /usr/lib/otter/container.ready
	touch /usr/lib/otter/container.bootstrapped
	printf "container_setup_done\n"

	# Fallback to standard init path, this is useful in case of non-Systemd containers
	# like an openrc alpine
	exec /sbin/init
}