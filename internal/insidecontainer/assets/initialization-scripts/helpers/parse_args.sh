# show_help will print usage to stdout.
# Arguments:
#   None
# Expected global variables:
#   version: otter version
# Expected env variables:
#   USER
#   HOME
# Outputs:
#   print usage with examples.
show_help()
{
	cat << EOF
╭────────────────────────────────────────────────────────────────╮
│▸ Options:                                                      │
│ --                    ┆ -e   Command to execute during init    │
│ --name                ┆ -n   User name                         │
│ --user                ┆ -u   UID of the user                   │
│ --home                ┆ -d   Path to the user's home directory │
│ --init                ┆ -I   Whether to use init or not        │
│ --group               ┆ -g   GID of the user                   │
│ --nvidia                     Integrate host's Nvidia drivers   │
│ --upgrade             ┆ -U   Run init in upgrade mode          │
│ --rootful                    Mark container as rootful         │
│ --pre-init-hooks             Commands to execute in pre-setup  │
│ --additional-packages        Extra packages to install         │
├────────────────────────────────────────────────────────────────┤
│▸ Extras:                                                       │
│ --help    ┆ -h  Show this help message                         │
│ --verbose ┆ -v  Show more verbose output                       │
│ --version ┆ -V  Show version information                       │
├────────────────────────────────────────────────────────────────┤
│▸ See also:                                                     │
│  otter-init is intended for internal use only, never           │
│  meant to be called directly. Its purpose is to set            │
│  up containers & handle upgrades.                              │
╰────────────────────────────────────────────────────────────────╯
EOF
}

# parse_args parses otter-init's CLI flags, setting the corresponding
# global variables as a side effect. Calls show_help/exits directly for
# --help, --version, and invalid flags.
# Arguments:
#   "$@": otter-init's positional parameters, forwarded as-is
# Expected global variables:
#   version: otter version
# Expected env variables:
#   None
# Outputs:
#   Sets: verbose, upgrade, rootful, container_user_name, init,
#   container_user_home, container_user_uid, container_user_gid,
#   pre_init_hook, container_additional_packages, nvidia, init_hook
parse_args()
{
	while :; do
		case $1 in
			-h | --help)
				# Call a "show_help" function to display a synopsis, then exit.
				show_help
				exit 0
				;;
			-v | --verbose)
				shift
				verbose=1
				;;
			-V | --version)
				printf "otter: %s\n" "${version}"
				exit 0
				;;
			-U | --upgrade)
				shift
				upgrade=1
				;;
			--rootful)
				shift
				rootful=1
				;;
			-n | --name)
				if [ -n "$2" ]; then
					container_user_name="$2"
					shift
					shift
				fi
				;;
			-i | --init)
				if [ -n "$2" ]; then
					init="$2"
					shift
					shift
				fi
				;;
			-d | --home)
				if [ -n "$2" ]; then
					container_user_home="$2"
					shift
					shift
				fi
				;;
			-u | --user)
				if [ -n "$2" ]; then
					container_user_uid="$2"
					shift
					shift
				fi
				;;
			-g | --group)
				if [ -n "$2" ]; then
					container_user_gid="$2"
					shift
					shift
				fi
				;;
			--pre-init-hooks)
				if [ -n "$2" ]; then
					pre_init_hook="$2"
				fi
				shift
				shift
				;;
			--additional-packages)
				if [ -n "$2" ]; then
					container_additional_packages="$2"
				fi
				shift
				shift
				;;
			--nvidia)
				if [ -n "$2" ]; then
					nvidia="$2"
					shift
					shift
				fi
				;;
			--)
				shift
				init_hook=$*
				break
				;;
			-*) # Invalid options.
				printf >&2 "Error: Invalid flag '%s'\n\n" "$1"
				show_help
				exit 1
				;;
			*) # Default case: If no more options then break out of the loop.
				break ;;
		esac
	done
}
