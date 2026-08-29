if status --is-interactive
	test -z "$USER" && set -gx USER (id -un 2> /dev/null)
	test -z "$UID"  && set -gx UID (id -ur 2> /dev/null)
	test -z "$EUID" && set -gx EUID (id -u  2> /dev/null)
	set -gx SHELL (getent passwd "$USER" | cut -f 7 -d :)

	# Append the host's PATH (passed in as HOST_PATH, since PATH itself is
	# left untouched on entry so the container's own package manager stays
	# reachable -- see BuildCommandArgs's getent/cut shell resolution,
	# which needs the container's native PATH, not a host-overridden one)
	# onto the container's PATH, skipping any entry already present so
	# host CLI tools stay reachable without ever taking priority over the
	# container's own.
	if test -n "$HOST_PATH"
		for host_path_entry in (string split : -- "$HOST_PATH")
			if not contains -- "$host_path_entry" $PATH
				set -gx PATH $PATH "$host_path_entry"
			end
		end
	end

	test -z "$XDG_RUNTIME_DIR"; and set -gx XDG_RUNTIME_DIR /run/user/(id -ru)
	test -z "$DBUS_SESSION_BUS_ADDRESS"; and set -gx DBUS_SESSION_BUS_ADDRESS unix:path=/run/user/(id -ru)/bus

	# Ensure we have these two variables from the host, so that graphical apps
	# also work in case we use a login session
	if test -z $XAUTHORITY
		set -gx XAUTHORITY (host-spawn --cwd / sh -c "printf "%s" \$XAUTHORITY" 2>/dev/null)
		# if the variable is still empty, unset it, because empty it could be harmful
		test -z $XAUTHORITY ; and set -e XAUTHORITY
	end
	if test -z $XAUTHLOCALHOSTNAME
		set -gx XAUTHLOCALHOSTNAME (host-spawn --cwd / sh -c "printf "%s" \$XAUTHLOCALHOSTNAME" 2>/dev/null)
		test -z $XAUTHLOCALHOSTNAME ; and set -e XAUTHLOCALHOSTNAME
	end
	if test -z $WAYLAND_DISPLAY
		set -gx WAYLAND_DISPLAY (host-spawn --cwd / sh -c "printf "%s" \$WAYLAND_DISPLAY" 2>/dev/null)
		test -z $WAYLAND_DISPLAY ; and set -e WAYLAND_DISPLAY
	end
	if test -z $DISPLAY
		set -gx DISPLAY (host-spawn --cwd / sh -c "printf "%s" \$DISPLAY" 2>/dev/null)
		test -z $DISPLAY ; and set -e DISPLAY
	end

	# This will ensure we have a first-shell password setup for an user if needed.
	# We're going to use this later in case of rootful containers
	if test -e /var/tmp/.$USER.passwd.initialize
		echo "⚠️  First time user password setup ⚠️ "
		trap "echo; exit" INT
		passwd && rm -f /var/tmp/.$USER.passwd.initialize
		trap - INT
	end
end
