# always TERM to xterm-256color
$env.TERM = "xterm-256color"

if (is-terminal --stdin) {
	if ($env.USER? | is-empty) { $env.USER = (id -un | str trim) }
	if ($env.UID? | is-empty) { $env.UID = (id -ur | str trim) }
	if ($env.EUID? | is-empty) { $env.EUID = (id -u | str trim) }
	# passwd's 7th field is the user's login shell. This mirrors the
	# `cut -f 7` used by posix_integration.sh / fish_integration.fish, but
	# nushell's `split column` names columns 0-based, so the shell is column6.
	$env.SHELL = (^getent passwd $env.USER | split column ':' | get column6.0)

	# Append the host's PATH (passed in as HOST_PATH, since PATH itself is
	# left untouched on entry so the container's own package manager stays
	# reachable -- see BuildCommandArgs's getent/cut shell resolution,
	# which needs the container's native PATH, not a host-overridden one)
	# onto the container's PATH, skipping any entry already present so
	# host CLI tools stay reachable without ever taking priority over the
	# container's own.
	if not ($env.HOST_PATH? | is-empty) {
		for host_path_entry in ($env.HOST_PATH | split row ':') {
			if not ($host_path_entry in $env.PATH) {
				$env.PATH = ($env.PATH | append $host_path_entry)
			}
		}
	}

	if ($env.XDG_RUNTIME_DIR? | is-empty) {
		$env.XDG_RUNTIME_DIR = $"/run/user/(id -ru | str trim)"
	}
	if ($env.DBUS_SESSION_BUS_ADDRESS? | is-empty) {
		$env.DBUS_SESSION_BUS_ADDRESS = $"unix:path=/run/user/(id -ru | str trim)/bus"
	}

	# Ensure we have these two variables from the host, so that graphical apps
	# also work in case we use a login session
	if ($env.XAUTHORITY? | is-empty) {
		let value = (host-spawn --cwd / sh -c 'printf "%s" $XAUTHORITY' | complete | get stdout | str trim)
		if not ($value | is-empty) { $env.XAUTHORITY = $value }
	}
	if ($env.XAUTHLOCALHOSTNAME? | is-empty) {
		let value = (host-spawn --cwd / sh -c 'printf "%s" $XAUTHLOCALHOSTNAME' | complete | get stdout | str trim)
		if not ($value | is-empty) { $env.XAUTHLOCALHOSTNAME = $value }
	}
	if ($env.WAYLAND_DISPLAY? | is-empty) {
		let value = (host-spawn --cwd / sh -c 'printf "%s" $WAYLAND_DISPLAY' | complete | get stdout | str trim)
		if not ($value | is-empty) { $env.WAYLAND_DISPLAY = $value }
	}
	if ($env.DISPLAY? | is-empty) {
		let value = (host-spawn --cwd / sh -c 'printf "%s" $DISPLAY' | complete | get stdout | str trim)
		if not ($value | is-empty) { $env.DISPLAY = $value }
	}

	# This will ensure we have a first-shell password setup for an user if needed.
	# We're going to use this later in case of rootful containers
	if (('/var/tmp/.' + $env.USER + '.passwd.initialize') | path exists) {
		print "⚠️  First time user password setup ⚠️ "
		passwd
		rm -f $'/var/tmp/.($env.USER).passwd.initialize'
	}
}
