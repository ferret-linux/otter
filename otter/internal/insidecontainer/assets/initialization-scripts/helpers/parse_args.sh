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

# require_value validates the argument following a value-taking flag.
# Arguments:
#   $1: flag name (used in error messages)
#   $2: number of remaining positional parameters
#   $3: the value (may be empty)
#   $4: 1 to also reject an empty value, 0 to allow one
# Outputs:
#   Prints an error and exits 1 when the value is absent or, when $4 is
#   1, empty. Optional flags can legitimately receive an empty value
#   (the host always passes --pre-init-hooks/--additional-packages).
require_value()
{
	if [ "$2" -lt 2 ]; then
		printf >&2 "Error: Missing value for '%s'\n\n" "$1"
		show_help
		exit 1
	fi
	if [ "$4" -eq 1 ] && [ -z "$3" ]; then
		printf >&2 "Error: Empty value for '%s'\n\n" "$1"
		show_help
		exit 1
	fi
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
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				verbose=1
				;;
			-V | --version)
				# shellcheck disable=SC2154 # assigned by otter-init before sourcing this file
				printf "otter: %s\n" "${version}"
				exit 0
				;;
			-U | --upgrade)
				shift
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				upgrade=1
				;;
			--rootful)
				shift
				# shellcheck disable=SC2034 # consumed by otter-init and helpers/setup_user.sh after sourcing this file
				rootful=1
				;;
			-n | --name)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init and helpers/setup_user.sh, helpers/setup_host_mounts.sh after sourcing this file
				container_user_name="$2"
				shift 2
				;;
			-i | --init)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				init="$2"
				shift 2
				;;
			-d | --home)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init and helpers/setup_user.sh after sourcing this file
				container_user_home="$2"
				shift 2
				;;
			-u | --user)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init and helpers/setup_user.sh after sourcing this file
				container_user_uid="$2"
				shift 2
				;;
			-g | --group)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init and helpers/setup_user.sh after sourcing this file
				container_user_gid="$2"
				shift 2
				;;
			--pre-init-hooks)
				require_value "$1" "$#" "$2" 0
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				pre_init_hook="$2"
				shift 2
				;;
			--additional-packages)
				require_value "$1" "$#" "$2" 0
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				container_additional_packages="$2"
				shift 2
				;;
			--nvidia)
				require_value "$1" "$#" "$2" 1
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
				nvidia="$2"
				shift 2
				;;
			--)
				shift
				# shellcheck disable=SC2034 # consumed by otter-init after sourcing this file
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
