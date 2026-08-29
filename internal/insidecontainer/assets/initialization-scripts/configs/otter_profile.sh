# always TERM to xterm-256color
export TERM=xterm-256color
###############################
if [ -z "${USER}" ]; then
	USER="$(id -un 2> /dev/null)"
	export USER
fi
if [ -z "${UID}" ]; then
	UID="$(id -ur 2> /dev/null)"
	readonly UID
fi
if [ -z "${EUID}" ]; then
	EUID="$(id -u  2> /dev/null)"
	readonly EUID
fi
SHELL="$(getent passwd "${USER}" | cut -f 7 -d :)"
export SHELL

# Append the host's PATH (passed in as HOST_PATH, since PATH itself is left
# untouched on entry so the container's own package manager stays reachable
# -- see BuildCommandArgs's getent/cut shell resolution above, which needs
# the container's native PATH, not a host-overridden one) onto the
# container's PATH, skipping any entry already present so host CLI tools
# stay reachable without ever taking priority over the container's own.
if [ -n "${HOST_PATH:-}" ]; then
	old_ifs="${IFS}"
	IFS=:
	for host_path_entry in ${HOST_PATH}; do
		case ":${PATH}:" in
			*":${host_path_entry}:"*) ;;
			*) PATH="${PATH}:${host_path_entry}" ;;
		esac
	done
	IFS="${old_ifs}"
	export PATH
fi

if [ -z "${XDG_RUNTIME_DIR:-}" ]; then
	XDG_RUNTIME_DIR="/run/user/$(id -ru)"
	export XDG_RUNTIME_DIR
fi
if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
	DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -ru)/bus"
	export DBUS_SESSION_BUS_ADDRESS
fi

# Ensure we have these two variables from the host, so that graphical apps
# also work in case we use a login session
if [ -z "${XAUTHORITY}" ]; then
    # shellcheck disable=SC2140 # inherited from distrobox: adjacent quotes intentionally concatenate into `printf %s $XAUTHORITY` for the inner sh -c
    XAUTHORITY="$(host-spawn --cwd / sh -c "printf "%s" \$XAUTHORITY" 2>/dev/null)"
    export XAUTHORITY
    # if the variable is still empty, unset it, because empty it could be harmful
    [ -z "${XAUTHORITY}" ] && unset XAUTHORITY
fi
if [ -z "${XAUTHLOCALHOSTNAME}" ]; then
    # shellcheck disable=SC2140 # inherited from distrobox: adjacent quotes intentionally concatenate into `printf %s $XAUTHLOCALHOSTNAME` for the inner sh -c
    XAUTHLOCALHOSTNAME="$(host-spawn --cwd / sh -c "printf "%s" \$XAUTHLOCALHOSTNAME" 2>/dev/null)"
    export XAUTHLOCALHOSTNAME
    [ -z "${XAUTHLOCALHOSTNAME}" ] && unset XAUTHLOCALHOSTNAME
fi
if [ -z "${WAYLAND_DISPLAY}" ]; then
    # shellcheck disable=SC2140 # inherited from distrobox: adjacent quotes intentionally concatenate into `printf %s $WAYLAND_DISPLAY` for the inner sh -c
    WAYLAND_DISPLAY="$(host-spawn --cwd / sh -c "printf "%s" \$WAYLAND_DISPLAY" 2>/dev/null)"
    export WAYLAND_DISPLAY
    [ -z "${WAYLAND_DISPLAY}" ] && unset WAYLAND_DISPLAY
fi
if [ -z "${DISPLAY}" ]; then
    # shellcheck disable=SC2140 # inherited from distrobox: adjacent quotes intentionally concatenate into `printf %s $DISPLAY` for the inner sh -c
    DISPLAY="$(host-spawn --cwd / sh -c "printf "%s" \$DISPLAY" 2>/dev/null)"
    export DISPLAY
    [ -z "${DISPLAY}" ] && unset DISPLAY
fi

# This will ensure a default prompt for a container, this will be remineshent of
# toolbx prompt: https://github.com/containers/toolbox/blob/main/profile.d/toolbox.sh#L47
# this will ensure greater compatibility between the two implementations
if [ -f /run/.toolboxenv ]; then
    # shellcheck disable=SC2154 # CONTAINER_ID is injected into the container's environment externally, not assigned in this script
    if [ "${BASH_VERSION:-}" != "" ]; then
        PS1="\[\e[0;90m\]╭⟮\[\e[0;92m\]⬢\[\e[0;90m\]⟯─⟮\[\e[0;96m\]\u\[\e[0;90m\]@\[\e[0;95m\]${CONTAINER_ID}\[\e[0;90m\]⟯─⟮\[\e[0;92m\]\W\[\e[0;90m\]⟯─⟮\[\e[0;94m\]$(date +%H:%M)\[\e[0;90m\]⟯\[\e[0m\]\n\[\e[0;90m\]╰─\[\e[0;92m\]▶ \[\e[0m\]"
        export PS1
    fi
    # shellcheck disable=SC2154 # CONTAINER_ID is injected into the container's environment externally, not assigned in this script
    # shellcheck disable=SC3003 # $'\n' is only ever evaluated under zsh (guarded above), which supports this syntax natively
    if [ "${ZSH_VERSION:-}" != "" ]; then
        PS1="%F{7}╭⟮%F{10}⬢%F{7}⟯─⟮%F{14}%n%F{7}@%F{13}${CONTAINER_ID}%F{7}⟯─⟮%F{10}%1~%F{7}⟯─⟮%F{12}%D{%H:%M}%F{7}⟯%f"$'\n'"%F{7}╰─%F{10}▶ %f"
        export PS1
    fi
fi

# This will ensure we have a first-shell password setup for an user if needed.
# We're going to use this later in case of rootful containers
if [ -e "/var/tmp/.${USER}.passwd.initialize" ]; then
	echo "⚠️  First time user password setup ⚠️ "
	trap "echo; exit" INT
	passwd && rm -f "/var/tmp/.${USER}.passwd.initialize"
	trap - INT
fi
