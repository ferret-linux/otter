# starship becomes the default prompt on official images only when ALL of the
# following hold:
#   - /usr/lib/otter/container.official exists
#   - the starship binary is on PATH
#   - /usr/lib/otter/helpers/starship.toml exists
#   - OTTER_DISABLE_STARSHIP_INTEGRATION is unset (setting it to any value —
#     even empty — opts out)
#   - /usr/lib/otter/container.no-starship does not exist
# There are two supported opt-out routes:
#   - temporary, per-enter:  otter enter --add-env OTTER_DISABLE_STARSHIP_INTEGRATION=1
#   - persistent, per-image: root shell, then  touch /usr/lib/otter/container.no-starship
# When opted out (or on a non-official image) the toolbox-style otter prompt
# below is used instead. otter-init unconditionally creates /run/.toolboxenv
# inside every otter container, so that marker alone selects this fallback on
# both official and non-official images.
#
# The otter starship config is set on the shell (not baked into the image)
# so the user's own config wins the moment the otter prompt is opted out of.
if [ -f /usr/lib/otter/container.official ] \
	&& command -v starship >/dev/null 2>&1 \
	&& [ -f /usr/lib/otter/helpers/starship.toml ] \
	&& [ "${OTTER_DISABLE_STARSHIP_INTEGRATION+x}" != "x" ] \
	&& [ ! -f /usr/lib/otter/container.no-starship ]; then
	export STARSHIP_CONFIG=/usr/lib/otter/helpers/starship.toml
	if [ "${BASH_VERSION:-}" != "" ]; then
		eval "$(starship init bash)"
	elif [ "${ZSH_VERSION:-}" != "" ]; then
		eval "$(starship init zsh)"
	fi
elif [ -f /run/.toolboxenv ]; then
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
