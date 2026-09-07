# The nushell analogue of fish_prompt.fish (and ps1_prompt.sh's PS1 for
# bash/zsh): a two-line, toolbox-style prompt showing the ⬢ otter mark, the
# user@container, the current directory and the clock, over a muted second
# line ending in the ▶ indicator.
#
# starship becomes the default prompt only when ALL of the following hold
# (see ps1_prompt.sh / fish_prompt.fish for the shared contract):
#   - /usr/lib/otter/container.official exists
#   - the starship binary is on PATH
#   - /usr/lib/otter/helpers/starship.toml exists
#   - OTTER_DISABLE_STARSHIP_INTEGRATION is unset (presence-based: setting it to
#     any value — even empty — opts out)
#   - /usr/lib/otter/container.no-starship does not exist
# Temporary per-enter opt-out:  otter enter --add-env OTTER_DISABLE_STARSHIP_INTEGRATION=1
# Persistent per-image opt-out:  touch /usr/lib/otter/container.no-starship
# When opted out, the otter prompt below is used instead.
#
# Installed by otter-init into /usr/local/share/nushell/vendor/autoload/, so
# it only exists inside otter containers.
if (is-terminal --stdin) {
	if (
		("/usr/lib/otter/container.official" | path exists)
		and (which starship | is-not-empty)
		and ("/usr/lib/otter/helpers/starship.toml" | path exists)
		and ($env.OTTER_DISABLE_STARSHIP_INTEGRATION? == null)
		and (not ("/usr/lib/otter/container.no-starship" | path exists))
	) {
		# The documented nushell & starship integration: generate starship's
		# nu init straight into nushell's vendor-autoload dir, which nushell
		# sources at startup (analogous to `starship init fish | source`).
		# No manual `source` needed. The otter starship config is set on the
		# shell (not baked into the image) so a user's own config wins the
		# moment the otter prompt is opted out of.
		$env.STARSHIP_CONFIG = "/usr/lib/otter/helpers/starship.toml"
		mkdir ($nu.data-dir | path join "vendor/autoload")
		starship init nu | save -f ($nu.data-dir | path join "vendor/autoload/starship.nu")
	} else {
		# Anything else — non-official image or starship opted out — uses the
		# toolbox-style otter prompt below, so a prompt is always shown. Drop
		# any previously generated starship autoload so a prior starship
		# session can't re-assert itself. `STARSHIP_CONFIG` is never set
		# here, so starship (if present) uses the user's own config.
		rm -f ($nu.data-dir | path join "vendor/autoload/starship.nu")
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