# Official otter images ship starship, so on those starship becomes the
# default prompt (see ps1_prompt.sh / nushell_prompt.nu). Non-official
# (user-built) otter images keep the toolbox-style otter prompt below.
if test -f /usr/lib/otter/container.official
	command -q starship; and starship init fish | source
else
	function fish_prompt
		set t (date +%H:%M)
		set dir (basename $PWD)
		echo -s (set_color brblack)"╭⟮"(set_color green)"⬢"(set_color brblack)"⟯─⟮"(set_color cyan)"$USER"(set_color brblack)"@"(set_color magenta)"$CONTAINER_ID"(set_color brblack)"⟯─⟮"(set_color green)"$dir"(set_color brblack)"⟯─⟮"(set_color blue)"$t"(set_color brblack)"⟯"(set_color normal)
		echo -s (set_color brblack)"╰─"(set_color green)"▶ "(set_color normal)
	end
end
