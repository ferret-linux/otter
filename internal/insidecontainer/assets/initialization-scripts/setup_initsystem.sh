# setup_initsystem fires up the container's init system (systemd, OpenRC,
# sysvinit, runit, dinit, or generic /sbin/init), masking host-conflicting
# units and setting up console/user integration first. Detection happens
# here; each supported init system does its own setup/launch in a dedicated
# setup_init_<name> function.
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
	# script(1) runs its -c command through $SHELL; override it to a plain
	# POSIX shell here so this internal, non-interactive use doesn't spawn
	# the container user's shell (e.g. fish) as root, which otherwise creates
	# stray root-owned ~/.cache, ~/.config, ~/.local state for no reason.
	SHELL=/bin/sh script -c "cat /var/console" /dev/null &

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
		printf "systemd" > /usr/lib/otter/container.initsystem
		setup_init_systemd
	elif [ -e /usr/sbin/openrc ] || [ -e /sbin/openrc ]; then
		printf "openrc" > /usr/lib/otter/container.initsystem
		setup_init_openrc
	elif [ -e /usr/sbin/telinit ] || [ -e /sbin/telinit ]; then
		# setup_init_sysvinit writes container.initsystem itself, once it
		# knows whether this is the LSB or BSD sysvinit layout.
		setup_init_sysvinit
	elif [ -e /usr/bin/runit ] || [ -e /usr/sbin/runit ] || [ -e /sbin/runit ]; then
		printf "runit" > /usr/lib/otter/container.initsystem
		setup_init_runit
	elif [ -e /usr/bin/dinit ] || [ -e /usr/sbin/dinit ] || [ -e /sbin/dinit ]; then
		printf "dinit" > /usr/lib/otter/container.initsystem
		setup_init_dinit
	elif [ -e /sbin/init ]; then
		printf "generic" > /usr/lib/otter/container.initsystem
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
mkdir -p /run/user/\$(id -ru)
chmod 700 /run/user/\$(id -ru)
ln -sf /run/host/run/user/\$(id -ru)/wayland-* /run/user/\$(id -ru)/
ln -sf /run/host/run/user/\$(id -ru)/pipewire-* /run/user/\$(id -ru)/
ln -sf /run/host/run/user/\$(id -ru)/bus /run/user/\$(id -ru)/bus
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

# write_resync_loop_script generates the shared
# /usr/local/bin/user-integration-resync script: a resident loop that
# blindly re-runs /usr/local/bin/user-integration every 60 seconds. Used
# only by the init systems with no native repeat/supervision primitive of
# their own (openrc, sysvinit-lsb, sysvinit-bsd, runit); systemd and dinit
# each re-trigger user-integration via their own native mechanisms instead
# (a .timer unit, and restart=true, respectively) and don't call this
# script.
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, passed through to
#     user-integration on each loop iteration
# Expected env variables:
#   None
# Outputs:
#   None
write_resync_loop_script()
{
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	cat << EOF > /usr/local/bin/user-integration-resync
#!/bin/sh
setsid su - "${container_user_name}" -c '
    while true; do
        /usr/local/bin/user-integration
        sleep 60
    done
'
EOF

	chmod +x /usr/local/bin/user-integration-resync
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

		# logind is masked by default in some images, but user@.service's PAM
		# session registration depends on it: without it, $XDG_RUNTIME_DIR
		# never gets set correctly for the user's systemd instance, which
		# causes it to create some of its own state (e.g. ~/.cache,
		# ~/.local/share) as root instead of as the mapped user.
		systemctl unmask systemd-logind.service dbus-org.freedesktop.login1.service || :
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

		# Periodically re-fires the oneshot service above, so integration
		# survives user-runtime-dir@.service wiping it mid-session and
		# picks up any host socket that appears after boot.
		cat << EOF > /usr/lib/systemd/system/user-integration@.timer
[Unit]
Description=Periodic user runtime integration resync for UID %i

[Timer]
OnBootSec=60s
OnUnitActiveSec=60s

[Install]
WantedBy=timers.target
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
	systemctl start user-integration@${container_user_name}.timer && \
	loginctl enable-linger ${container_user_name} || : && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	[ -e /usr/lib/systemd/systemd ] && exec /usr/lib/systemd/systemd --system --log-target=console --unit=multi-user.target
	[ -e /lib/systemd/systemd ] && exec /lib/systemd/systemd --system --log-target=console --unit=multi-user.target
}

# setup_init_sysvinit disables host-conflicting getty respawns, then
# dispatches to the real sysvinit layout in use: LSB-style (Devuan) or
# BSD-style (Slackware — detected via /etc/rc.d/rc.M's real presence, not
# by distro). Both run the same upstream sysvinit package, so readiness
# ("runlevel" command) is identical for either; only the service-script
# layout differs.
# Arguments:
#   None
# Expected global variables:
#   None
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

	if [ -e /etc/rc.d/rc.M ]; then
		setup_init_sysvinit_bsd
	else
		setup_init_sysvinit_lsb
	fi
}

# wait_for_sysvinit_runlevel blocks until sysvinit reports a real runlevel,
# then marks the container ready. Shared by both sysvinit layouts.
# Arguments:
#   None
# Expected global variables:
#   None
# Expected env variables:
#   None
# Outputs:
#   None
wait_for_sysvinit_runlevel()
{
	# "runlevel" prints "<previous> <current>", e.g. "N 2"; it prints
	# "unknown" for both fields before the switch happens.
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    current_runlevel=\$(runlevel | awk '{print \$2}'); \
	    [ \"\${current_runlevel}\" != \"unknown\" ] && [ -n \"\${current_runlevel}\" ] && break; \
		echo 'waiting for sysvinit to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &
}

# setup_init_sysvinit_lsb handles the LSB-style layout (Devuan).
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the user-integration
#     LSB script
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_sysvinit_lsb()
{
	write_user_integration_script
	write_resync_loop_script

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
		/usr/local/bin/user-integration-resync &
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

	printf "sysvinit-lsb" > /usr/lib/otter/container.initsystem

	wait_for_sysvinit_runlevel

	exec /sbin/init
}

# setup_init_sysvinit_bsd handles the BSD-style layout (Slackware). Unlike
# LSB, scripts under /etc/init.d don't autostart here — the documented
# mechanism is a native /etc/rc.d/rc.<name> script invoked from rc.local.
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the user-integration
#     rc script
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_sysvinit_bsd()
{
	write_user_integration_script
	write_resync_loop_script

	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	cat << EOF > /etc/rc.d/rc.otter-user-integration
#!/bin/sh

case "\$1" in
	start)
		/usr/local/bin/user-integration-resync &
		;;
	stop | restart | status)
		;;
	*)
		echo "Usage: \$0 start" >&2
		exit 1
		;;
esac

exit 0
EOF

	chmod +x /etc/rc.d/rc.otter-user-integration

	# Append once, guarded by a marker, so a container restart doesn't
	# duplicate the line or clobber any existing rc.local content.
	[ -e /etc/rc.d/rc.local ] || touch /etc/rc.d/rc.local
	if ! grep -qF "# otter-user-integration" /etc/rc.d/rc.local 2> /dev/null; then
		printf '\n# otter-user-integration\n[ -x /etc/rc.d/rc.otter-user-integration ] && /etc/rc.d/rc.otter-user-integration start\n' >> /etc/rc.d/rc.local
	fi
	chmod +x /etc/rc.d/rc.local

	printf "sysvinit-bsd" > /usr/lib/otter/container.initsystem

	wait_for_sysvinit_runlevel

	exec /sbin/init
}

# setup_init_openrc handles OpenRC regardless of which process actually owns
# PID 1. Detected via the "openrc" binary's existence (both /usr/sbin and
# /sbin, for merged- and non-merged-/usr layouts) rather than by distro, since
# OpenRC is layered on top of another init rather than being one itself.
# Concretely, two real cases exist and are handled internally rather than as
# separate dispatch branches:
#   - "openrc-init" is present: OpenRC's own real init binary is PID 1 (it can
#     be launched directly or symlinked to /sbin/init).
#   - "openrc-init" is absent: some other program (e.g. busybox-init, or
#     sysvinit as on default Gentoo) is PID 1 and hands off to OpenRC via
#     "openrc sysinit"/"boot"/"default" calls from its own inittab. This is
#     detected here, not by distro, because the /sbin/init this case launches
#     is a foreign binary we don't own and can't distro-match reliably.
# Either way, OpenRC itself does the same service management (dependency
# resolution, /etc/init.d, "default" runlevel), so integration and readiness
# are handled identically; only the final exec target differs.
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the user-integration
#     openrc-run script
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_openrc()
{
	write_user_integration_script
	write_resync_loop_script

	# openrc-run scripts must use this shebang: it's a real shell wrapper
	# (see openrc's sh/openrc-run.sh.in) that parses the "--lockfd <fd> start"
	# arguments openrc's exec_service() actually invokes services with — a
	# plain "case "$1" in start)" LSB-style script (as used for sysvinit)
	# would see "--lockfd" as $1 and never match.
	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	cat << EOF > /etc/init.d/otter-user-integration
#!/sbin/openrc-run
description="Otter host user-session integration"

depend()
{
	after *
	keyword -timeout
}

start()
{
	/usr/local/bin/user-integration-resync &
}
EOF

	chmod +x /etc/init.d/otter-user-integration

	if command -v rc-update > /dev/null 2>&1; then
		rc-update add otter-user-integration default > /dev/null 2>&1 || :
	fi

	# Wait for OpenRC to finish reaching the "default" runlevel before
	# marking the container ready. OpenRC writes the active runlevel's name
	# to /run/openrc/softlevel only after every service in it has started
	# (rc_runlevel_set() in librc runs after wait_for_services()), so this
	# is a genuine boot-complete signal, not a guess.
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    current_softlevel=\$(cat /run/openrc/softlevel 2> /dev/null); \
	    [ \"\${current_softlevel}\" = \"default\" ] && break; \
		echo 'waiting for openrc to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	if [ -e /usr/sbin/openrc-init ]; then
		exec /usr/sbin/openrc-init
	elif [ -e /sbin/openrc-init ]; then
		exec /sbin/openrc-init
	else
		# Some other program (busybox-init, sysvinit, ...) is PID 1 and is
		# the one that actually calls into openrc; just hand off to it.
		exec /sbin/init
	fi
}

# setup_init_runit handles Void's runit. Detected via the "runit" binary's
# existence (/usr/bin, plus /usr/sbin and /sbin for older/non-relocated
# layouts) rather than by distro, matching the systemd/openrc/sysvinit
# precedent. Void's own packaging relocates the real binary to
# /usr/bin/runit (void-packages runit template: "vsed -e
# 's,sbin/runit,usr/bin/runit,g' -i runit.h"); /sbin/init itself is a
# symlink to "runit-init", a tiny shim that, once it detects it's running
# as PID 1, immediately execve()s the real runit binary and never returns
# (void-linux/runit's runit-init.c: "if (getpid() == 1) { ... execve(RUNIT,
# ...) }"). We exec the resolved runit path directly instead of going
# through /sbin/init, skipping that redundant (and already a no-op) hop.
# Two known-shipped agetty services (agetty-console, agetty-serial,
# agetty-tty1, agetty-hvc0, agetty-hvsi0 - confirmed via runit-void's
# packaging template, which ships conf files for exactly these five and
# documents them as enabled by default) are explicitly disabled before
# boot, the same host-getty-conflict concern already handled for systemd
# (getty@.service masking) and sysvinit (inittab sed); only the five
# confirmed names are targeted, not a glob, to avoid touching any
# additional agetty service a user might add later via
# "--additional-packages". Readiness is polled via "sv status
# /var/service/*" - /var/service is runit's own documented live symlink to
# the active runsvdir (confirmed in runsvchdir(8): "Normally /var/service
# is a symlink to current") - guarded against not existing yet, since it's
# only created once stage 2 actually runs. User-session integration uses a
# oneshot run script that execs the "pause" binary Void ships specifically
# for this (a no-op signal-blocking process) after running the integration
# script once, to avoid runsv's normal respawn-on-exit behaviour; the
# service is statically enabled into runsvdir/default before exec, the
# same static enable-then-boot pattern already used for OpenRC/sysvinit
# (runit has no systemd-style lazy-instantiation to hook into instead).
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the
#     user-integration run script
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_runit()
{
	# Known agetty services shipped enabled-by-default by runit-void's
	# packaging (confirmed via its template's conf_files list). Removing
	# only the enable-symlink, not the service definition under /etc/sv -
	# fully reversible, and harmless if a given image never enabled them.
	rm -f /etc/runit/runsvdir/default/agetty-console \
		/etc/runit/runsvdir/default/agetty-serial \
		/etc/runit/runsvdir/default/agetty-tty1 \
		/etc/runit/runsvdir/default/agetty-hvc0 \
		/etc/runit/runsvdir/default/agetty-hvsi0

	write_user_integration_script
	write_resync_loop_script

	mkdir -p /etc/sv/otter-user-integration

	cat << EOF > /etc/sv/otter-user-integration/run
#!/bin/sh
exec /usr/local/bin/user-integration-resync
EOF

	chmod +x /etc/sv/otter-user-integration/run

	if [ -d /etc/runit/runsvdir/default ]; then
		ln -sf /etc/sv/otter-user-integration /etc/runit/runsvdir/default/otter-user-integration
	fi

	# Wait for runit's services to report "run" status before marking the
	# container ready. /var/service only exists once stage 2 has actually
	# run (it's created by "runsvchdir" via a symlink, not present from the
	# start), so the check tolerates it being briefly absent.
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    [ -e /var/service ] && ! sv status /var/service/* 2> /dev/null | grep -qv '^run:' && break; \
		echo 'waiting for runit services to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	if [ -e /usr/bin/runit ]; then
		exec /usr/bin/runit
	elif [ -e /usr/sbin/runit ]; then
		exec /usr/sbin/runit
	else
		exec /sbin/runit
	fi
}

# setup_init_dinit handles dinit (Chimera Linux, and Wolfi's from-source
# build - dinit is not distro-specific, so detection is by binary
# existence, same as the other three non-systemd inits). On both, the real
# binary lives at /usr/bin/dinit (Chimera: cports' own Packaging.md states
# "/usr/sbin, /bin and /sbin are symbolic links to the primary /usr/bin
# path and should never be present in packages", i.e. full usr-merge by
# policy; Wolfi: its Containerfile builds vanilla dinit from source and
# does "ln -sf /usr/bin/dinit /sbin/init" directly) - /usr/sbin and /sbin
# are still checked for defensiveness, matching the systemd/openrc/runit
# precedent, since dinit itself is not tied to either distro's layout.
# User-session integration is a "scripted" (run-once) dinit service,
# enabled the same static pre-boot way as OpenRC/sysvinit/runit: a symlink
# into /etc/dinit.d/boot.d (Chimera's own service docs, verbatim: "What
# this does is simply create a symlink in /etc/dinit.d/boot.d" - this is
# the documented, generic dinit enablement mechanism, not Chimera-specific
# tooling). The boot.d directory's existence is only checked, never
# created - on Chimera dinit-chimera already provides it; a from-scratch
# dinit setup (e.g. today's Wolfi image) has no "boot" service at all yet,
# which is a separate, already-flagged gap to be fixed in the Containerfile
# itself, not papered over here.
# Readiness is polled via "dinitctl status boot", empirically verified
# against a real dinit build (cloned and built davmac314/dinit from source
# in a sandbox, ran it against a live service tree matching this design):
# output is confirmed to be "State: STARTED" on a running/completed
# service, exit code 0 - not assumed from docs.
# No getty conflict exists from dinit-chimera itself: cloned
# chimera-linux/dinit-chimera directly and grepped its full, authoritative
# service list (services/meson.build - every file the package installs,
# 50 total) for getty/agetty - zero matches. "login.target" (previously
# suspected) is confirmed to be a plain synchronization point (type =
# internal, options: runs-on-console) that starts nothing itself.
# A conflict does become real if a third-party image additionally installs
# "nyagetty" (Chimera's standalone agetty), confirmed via its actual
# cports package (main/nyagetty/template.py): post_install() calls
# install_service(..., enable=True) on a milestone that dynamically spawns
# per-tty getty instances - so the milestone symlink is proactively
# removed above; chimera.Containerfile itself never installs nyagetty, so
# this is a no-op today and a real fix for images that do.
# The exec line passes "--container" (a real, documented dinit flag: "run
# in container mode (do not manage system)") - confirmed via "dinit
# --help" against the real build. This isn't needed for dinit to stay
# resident as PID 1 (verified separately with "unshare --pid": a bare
# exec with no flags, actually running as PID 1, correctly stays up as
# the persistent service manager - dinit auto-detects PID 1 same as
# runit-init did). It's added because, unlike systemd (masks host-only
# units via the "container=" env var already set by otter's docker.go/
# podman.go) and OpenRC/runit (self-detect via similar container checks),
# dinit has no equivalent auto-detection found in its source - "--container"
# is the explicit, documented way to tell it the same thing, so it's
# passed rather than relying on default (system-install-assuming) behaviour.
# Arguments:
#   None
# Expected global variables:
#   container_user_name: the container's primary user, for the
#     user-integration service's command
# Expected env variables:
#   None
# Outputs:
#   None
setup_init_dinit()
{
	# If a image installs "nyagetty" alongside dinit-chimera,
	# its package auto-enables this milestone (cports' own install_service
	# helper, called with enable=True, symlinks it into
	# /usr/lib/dinit.d/boot.d unconditionally at package-install time).
	# The milestone itself dynamically spawns per-tty getty instances on
	# demand rather than shipping them as fixed pre-enabled services, so
	# stopping this one symlink from running is sufficient to prevent all
	# of them - same host-getty-conflict concern already handled for
	# systemd/sysvinit/runit. Removing only the enable-symlink, not the
	# service definition at /usr/lib/dinit.d/agetty - fully reversible,
	# and a harmless no-op on today's chimera.Containerfile, which never
	# installs nyagetty.
	rm -f /usr/lib/dinit.d/boot.d/agetty

	write_user_integration_script

	# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
	cat << EOF > /etc/dinit.d/otter-user-integration
type = scripted
command = /bin/sh -c "su - ${container_user_name} -c /usr/local/bin/user-integration; sleep 60"
restart = true
EOF

	if [ -d /etc/dinit.d/boot.d ]; then
		ln -sf /etc/dinit.d/otter-user-integration /etc/dinit.d/boot.d/otter-user-integration
	fi

	# Wait for dinit's "boot" service (the universal root of the dependency
	# tree) to report started before marking the container ready.
	sh -c "timeout=120 && sleep 1 && while [ \"\${timeout}\" -gt 0 ]; do \
	    dinitctl status boot 2> /dev/null | grep -q 'State: STARTED' && break; \
		echo 'waiting for dinit to come up...\n' && sleep 1 && timeout=\$(( timeout -1 )); \
	done && \
	touch /usr/lib/otter/container.ready && \
	touch /usr/lib/otter/container.bootstrapped && \
	echo container_setup_done" &

	if [ -e /usr/bin/dinit ]; then
		exec /usr/bin/dinit --container
	elif [ -e /usr/sbin/dinit ]; then
		exec /usr/sbin/dinit --container
	else
		exec /sbin/dinit --container
	fi
}

# setup_init_generic is the last-resort fallback for any init that ships a
# standard /sbin/init entrypoint but isn't systemd, OpenRC, sysvinit, runit,
# or dinit (all of which are detected and handled by their own dedicated
# functions above).
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

	# Fallback to standard init path for any init system not specifically
	# detected above
	exec /sbin/init
}
