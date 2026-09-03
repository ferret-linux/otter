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