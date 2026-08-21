# always TERM to xterm-256color
export TERM=xterm-256color
###############################
test -z "$USER" && export USER="$(id -un 2> /dev/null)"
test -z "$UID"  && readonly UID="$(id -ur 2> /dev/null)"
test -z "$EUID" && readonly EUID="$(id -u  2> /dev/null)"
export SHELL="$(getent passwd "${USER}" | cut -f 7 -d :)"

test -z "${XDG_RUNTIME_DIR:-}" && export XDG_RUNTIME_DIR="/run/user/$(id -ru)"
test -z "${DBUS_SESSION_BUS_ADDRESS:-}" && export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -ru)/bus"

# Ensure we have these two variables from the host, so that graphical apps
# also work in case we use a login session
if [ -z "$XAUTHORITY" ]; then
    export XAUTHORITY="$(host-spawn sh -c "printf "%s" \$XAUTHORITY")"
    # if the variable is still empty, unset it, because empty it could be harmful
    [ -z "$XAUTHORITY" ] && unset XAUTHORITY
fi
if [ -z "$XAUTHLOCALHOSTNAME" ]; then
    export XAUTHLOCALHOSTNAME="$(host-spawn sh -c "printf "%s" \$XAUTHLOCALHOSTNAME")"
    [ -z "$XAUTHLOCALHOSTNAME" ] && unset XAUTHLOCALHOSTNAME
fi
if [ -z "$WAYLAND_DISPLAY" ]; then
    export WAYLAND_DISPLAY="$(host-spawn sh -c "printf "%s" \$WAYLAND_DISPLAY")"
    [ -z "$WAYLAND_DISPLAY" ] && unset WAYLAND_DISPLAY
fi
if [ -z "$DISPLAY" ]; then
    export DISPLAY="$(host-spawn sh -c "printf "%s" \$DISPLAY")"
    [ -z "$DISPLAY" ] && unset DISPLAY
fi

# This will ensure a default prompt for a container, this will be remineshent of
# toolbx prompt: https://github.com/containers/toolbox/blob/main/profile.d/toolbox.sh#L47
# this will ensure greater compatibility between the two implementations
if [ -f /run/.toolboxenv ]; then
    [ "${BASH_VERSION:-}" != "" ] && export PS1="\[\e[0;90m\]╭⟮\[\e[0;92m\]⬢\[\e[0;90m\]⟯─⟮\[\e[0;96m\]\u\[\e[0;90m\]@\[\e[0;95m\]$CONTAINER_ID\[\e[0;90m\]⟯─⟮\[\e[0;92m\]\W\[\e[0;90m\]⟯─⟮\[\e[0;94m\]$(date +%H:%M)\[\e[0;90m\]⟯\[\e[0m\]\n\[\e[0;90m\]╰─\[\e[0;92m\]▶ \[\e[0m\]"
    [ "${ZSH_VERSION:-}" != "" ] && export PS1="%F{7}╭⟮%F{10}⬢%F{7}⟯─⟮%F{14}%n%F{7}@%F{13}$CONTAINER_ID%F{7}⟯─⟮%F{10}%1~%F{7}⟯─⟮%F{12}%D{%H:%M}%F{7}⟯%f"$'\n'"%F{7}╰─%F{10}▶ %f"
fi

# This will ensure we have a first-shell password setup for an user if needed.
# We're going to use this later in case of rootful containers
if [ -e /var/tmp/.$USER.passwd.initialize ]; then
	echo "⚠️  First time user password setup ⚠️ "
	trap "echo; exit" INT
	passwd && rm -f /var/tmp/.$USER.passwd.initialize
	trap - INT
fi