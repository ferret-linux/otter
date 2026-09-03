# The nushell analogue of fish_prompt.fish (and ps1_prompt.sh's PS1 for
# bash/zsh): a two-line, toolbox-style prompt showing the ⬢ otter mark, the
# user@container, the current directory and the clock, over a muted second
# line ending in the ▶ indicator. Official otter images ship starship, so on
# those starship becomes the default prompt instead (see ps1_prompt.sh /
# fish_prompt.fish); this otter prompt is kept for non-official images.
#
# Installed by otter-init into /usr/local/share/nushell/vendor/autoload/, so
# it only exists inside otter containers -- no /run/.toolboxenv guard needed,
# unlike the bash profile which also ships to the host.
if (is-terminal --stdin) {
	if ("/usr/lib/otter/container.official" | path exists) {
		if (which starship | is-not-empty) {
			# Generate starship's nu init into the cache dir, then source it to
			# install its PROMPT_COMMAND override. nushell's `source` needs a
			# parse-time constant path, so the cache path must be a `const` and
			# existence is expressed as `const ... else { null }`.
			const starship_init = $"($nu.cache-dir)/starship/init.nu"
			mkdir (($starship_init | path dirname) | path expand)
			^starship init nu | save -f $starship_init
			const src = if ($starship_init | path expand | path exists) { $starship_init } else { null }
			source $src
		}
	} else {
		if ($env.CONTAINER_ID? | is-empty) {
			# otter normally injects CONTAINER_ID (see podman.go), but fall back
			# to the hostname so the prompt survives shells without it.
			$env.CONTAINER_ID = (hostname | str trim)
		}

		# PROMPT_COMMAND is re-evaluated before every prompt, so it reads the
		# live $env.USER / $env.PWD / clock, mirroring fish's dynamic prompt.
		#
		# Color codes mirror fish_prompt.fish exactly: the surface glyphs use
		# bright black (90) and the accents use the named codes. nushell doesn't
		# expose bright black by name, so it's emitted via `ansi -e '90m'`.
		$env.PROMPT_COMMAND = {||
			let t = (date now | format date '%H:%M')
			let dir = ($env.PWD | path basename)
			let user = ($env.USER? | default (id -un | str trim))
			(
				$"(ansi -e '90m')╭⟮(ansi green)⬢(ansi -e '90m')⟯─⟮(ansi cyan)($user)(ansi -e '90m')@(ansi magenta)($env.CONTAINER_ID)(ansi -e '90m')⟯─⟮(ansi green)($dir)(ansi -e '90m')⟯─⟮(ansi blue)($t)(ansi -e '90m')⟯(ansi reset)"
				+ (char newline)
				+ $"(ansi -e '90m')╰─(ansi green)▶ (ansi reset)"
			)
		}
	}
}