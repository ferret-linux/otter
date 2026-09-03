# starship becomes the default prompt only when ALL of the following hold
# (see ps1_prompt.sh / nushell_prompt.nu for the shared contract):
#   - /usr/lib/otter/container.official exists
#   - the starship binary is on PATH
#   - /usr/lib/otter/helpers/starship.toml exists
#   - OTTER_DISABLE_STARSHIP_INTEGRATION is unset (presence-based: setting it to
#     any value — even empty — opts out)
#   - /usr/lib/otter/container.no-starship does not exist
# Temporary per-enter opt-out:  otter enter --add-env OTTER_DISABLE_STARSHIP_INTEGRATION=1
# Persistent per-image opt-out:  touch /usr/lib/otter/container.no-starship
if test -f /usr/lib/otter/container.official
	and command -q starship
	and test -f /usr/lib/otter/helpers/starship.toml
	and not set -q OTTER_DISABLE_STARSHIP_INTEGRATION
	and not test -f /usr/lib/otter/container.no-starship
	starship init fish | source
else if test -f /run/.toolboxenv
	set -e STARSHIP_CONFIG
	function fish_prompt
		set t (date +%H:%M)
		set dir (basename $PWD)
		echo -s (set_color brblack)"╭⟮"(set_color green)"⬢"(set_color brblack)"⟯─⟮"(set_color cyan)"$USER"(set_color brblack)"@"(set_color magenta)"$CONTAINER_ID"(set_color brblack)"⟯─⟮"(set_color green)"$dir"(set_color brblack)"⟯─⟮"(set_color blue)"$t"(set_color brblack)"⟯"(set_color normal)
		echo -s (set_color brblack)"╰─"(set_color green)"▶ "(set_color normal)
	end
end
