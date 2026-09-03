#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./silence-shell.sh [<fish|nu> ...]
# Silences the interactive startup/welcome messages that fish and nushell
# print on launch, by dropping a vendor config into the same XDG_DATA_DIRS-
# driven directories otter-init populates at container-setup time:
#   - fish:     /usr/local/share/fish/vendor_conf.d     (fish_greeting)
#   - nushell:  /usr/local/share/nushell/vendor/autoload (show_banner)
# Defaults to silencing both when no shells are named. Idempotent, like
# python-fix.sh.

set -e

silence_fish()
{
	conf_dir="/usr/local/share/fish/vendor_conf.d"
	mkdir -p "${conf_dir}"

	# fish prints the greeting tip on interactive startup; an empty
	# fish_greeting universal variable suppresses it.
	cat > "${conf_dir}/silence-shell.fish" <<'EOF'
set -g fish_greeting
EOF
}

silence_nu()
{
	autoload_dir="/usr/local/share/nushell/vendor/autoload"
	mkdir -p "${autoload_dir}"

	# Nushell prints a welcome banner, an uptime report and a startup-time
	# measurement on interactive startup; config.show_banner=false disables
	# them. Guarded so it only applies to interactive sessions, matching
	# otter_config.nu.
	cat > "${autoload_dir}/silence-shell.nu" <<'EOF'
if (is-terminal --stdin) {
	$env.config.show_banner = false
}
EOF
}

if [ "$#" -eq 0 ]; then
	silence_fish
	silence_nu
	exit 0
fi

for requested in "$@"; do
	case "${requested}" in
		fish) silence_fish ;;
		nu) silence_nu ;;
		*) echo "ERROR: silence-shell.sh does not support '${requested}' (only fish, nu)" >&2; exit 1 ;;
	esac
done