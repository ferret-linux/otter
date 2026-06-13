#compdef otter
# zsh completion for otter
# Drop in $fpath, e.g. /usr/local/share/zsh/site-functions/_otter
# or ~/.local/share/zsh/site-functions/_otter (add dir to fpath first)

_otter_containers() {
    local -a containers
    containers=( ${(f)"$(otter list --json 2>/dev/null | grep '"name"' | sed 's/.*"name": *"\([^"]*\)".*/\1/')"} )
    echo "${containers[@]}"
}

_otter_registry_images() {
    local -a images
    images=( ${(f)"$(otter registry 2>/dev/null | awk 'NR>1 {print $1}')"} )
    echo "${images[@]}"
}

_otter() {
    local state line
    typeset -A opt_args

    local -a global_flags
    global_flags=(
        '(--root -r)'{--root,-r}'[run as root]'
        '(--help -h)'{--help,-h}'[show help]'
        '--version[show version]'
    )

    _arguments -C \
        "${global_flags[@]}" \
        '1: :_otter_subcommands' \
        '*:: :->subcmd' \
        && return

    case "${state}" in
        subcmd)
            case "${words[1]}" in
                assemble|dmf)        _otter_assemble ;;
                create|mk)           _otter_create ;;
                enter|sh)            _otter_enter ;;
                generate-entry|pin)  _otter_generate_entry ;;
                inspect|info)        _otter_inspect ;;
                journal|logs)        _otter_journal ;;
                list|ls)             _otter_list ;;
                lock|lk)             _otter_lock ;;
                pause|zz)            _otter_pause ;;
                registry|reg)        _otter_registry ;;
                remove|rm)           _otter_remove ;;
                restart|rbt)         _otter_restart ;;
                start|up)            _otter_start ;;
                stop|dn)             _otter_stop ;;
                unlock|ulk)          _otter_unlock ;;
                upgrade|syu)         _otter_upgrade ;;
            esac
            ;;
    esac
}

_otter_subcommands() {
    local -a subcmds
    subcmds=(
        'assemble:apply a manifest file (alias: dmf)'
        'create:create a new container (alias: mk)'
        'enter:enter a container shell (alias: sh)'
        'generate-entry:create a desktop entry (alias: pin)'
        'inspect:show container details (alias: info)'
        'journal:show container logs (alias: logs)'
        'list:list otter containers (alias: ls)'
        'lock:lock a container (alias: lk)'
        'pause:pause a container (alias: zz)'
        'registry:manage otter images (alias: reg)'
        'remove:remove a container (alias: rm)'
        'restart:restart a container (alias: rbt)'
        'start:start a container (alias: up)'
        'stop:stop a container (alias: dn)'
        'unlock:unlock a container (alias: ulk)'
        'upgrade:upgrade packages in a container (alias: syu)'
    )
    _describe 'command' subcmds
}

_otter_container_names() {
    local -a names
    names=( $(_otter_containers) )
    if [[ ${#names} -gt 0 ]]; then
        _values 'container' "${names[@]}"
    fi
}

_otter_assemble() {
    local state
    _arguments -C \
        '(--help -h)'{--help,-h}'[show help]' \
        '1: :_otter_assemble_subcommands' \
        '*:: :->assemble_sub' \
        && return

    case "${state}" in
        assemble_sub)
            case "${words[1]}" in
                create|mk)
                    _arguments \
                        '(--file -f)'{--file,-f}'[manifest file]:file:_files -g "*.toml"' \
                        '(--replace -R)'{--replace,-R}'[replace existing containers]' \
                        '(--help -h)'{--help,-h}'[show help]'
                    ;;
                remove|rm)
                    _arguments \
                        '(--file -f)'{--file,-f}'[manifest file]:file:_files -g "*.toml"' \
                        '(--help -h)'{--help,-h}'[show help]'
                    ;;
            esac
            ;;
    esac
}

_otter_assemble_subcommands() {
    local -a subcmds
    subcmds=(
        'create:create containers from manifest (alias: mk)'
        'remove:remove containers from manifest (alias: rm)'
    )
    _describe 'assemble subcommand' subcmds
}

_otter_create() {
    _arguments \
        '(--image -i)'{--image,-i}'[container image]:image:( $(_otter_registry_images) )' \
        '(--hostname -n)'{--hostname,-n}'[container hostname]:hostname' \
        '(--shell -s)'{--shell,-s}'[shell to use]:shell:(bash zsh fish)' \
        '(--pull -p)'{--pull,-p}'[always pull image]' \
        '(--clone -C)'{--clone,-C}'[clone from container]:container:( $(_otter_containers) )' \
        '(--home -H)'{--home,-H}'[custom home directory]:dir:_files -/' \
        '(--volume -v)'{--volume,-v}'[additional volume mounts]:volume' \
        '(--additional-flags -a)'{--additional-flags,-a}'[extra flags for container runtime]:flags' \
        '(--additional-packages -ap)'{--additional-packages,-ap}'[packages to install]:packages' \
        '(--init-hooks -ih)'{--init-hooks,-ih}'[post-init hook commands]:hook' \
        '(--pre-init-hooks -ph)'{--pre-init-hooks,-ph}'[pre-init hook commands]:hook' \
        '(--init -I)'{--init,-I}'[enable init system (systemd)]' \
        '(--memory -m)'{--memory,-m}'[memory limit (e.g. 512m, 2g)]:memory' \
        '(--cpu-threads -t)'{--cpu-threads,-t}'[cpu thread limit]:threads' \
        '(--nvidia -N)'{--nvidia,-N}'[enable nvidia GPU support]' \
        '(--platform -P)'{--platform,-P}'[container platform]:platform:(linux/amd64 linux/arm64 linux/arm/v7 linux/386)' \
        '(--unshare-devsys -ud)'{--unshare-devsys,-ud}'[unshare host devices and sysfs]' \
        '(--unshare-groups -ug)'{--unshare-groups,-ug}'[unshare host groups]' \
        '(--unshare-ipc -ui)'{--unshare-ipc,-ui}'[unshare host IPC namespace]' \
        '(--unshare-netns -un)'{--unshare-netns,-un}'[unshare host network namespace]' \
        '(--unshare-process -up)'{--unshare-process,-up}'[unshare host process namespace]' \
        '(--unshare-all -ua)'{--unshare-all,-ua}'[unshare all namespaces]' \
        '(--no-entry -E)'{--no-entry,-E}'[skip desktop entry generation]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]'
}

_otter_enter() {
    _arguments \
        '(--clean-path -c)'{--clean-path,-c}'[use clean standard PATH]' \
        '(--additional-flags -a)'{--additional-flags,-a}'[extra flags for exec]:flags' \
        '(--no-tty -T)'{--no-tty,-T}'[disable TTY allocation]' \
        '(--no-workdir -nw)'{--no-workdir,-nw}'[use home dir instead of cwd]' \
        '(--add-env -e)'{--add-env,-e}'[pass additional env vars]:env' \
        '(--empty-env -E)'{--empty-env,-E}'[start with empty environment]' \
        '(--auto-start -S)'{--auto-start,-S}'[start container if stopped]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_generate_entry() {
    _arguments \
        '(--delete -d)'{--delete,-d}'[remove desktop entry]' \
        '(--icon -i)'{--icon,-i}'[icon path or "auto"]:icon:_files' \
        '(--all -a)'{--all,-a}'[apply to all containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_inspect() {
    _arguments \
        '(--json -j)'{--json,-j}'[output as JSON]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_journal() {
    _arguments \
        '(--follow -f)'{--follow,-f}'[follow log output]' \
        '(--since -s)'{--since,-s}'[show logs since timestamp]:timestamp' \
        '(--until -u)'{--until,-u}'[show logs until timestamp]:timestamp' \
        '(--timestamps -t)'{--timestamps,-t}'[show timestamps]' \
        '(--tail -n)'{--tail,-n}'[number of lines to show]:lines' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_list() {
    _arguments \
        '(--json -j)'{--json,-j}'[output as JSON]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]'
}

_otter_lock() {
    _arguments \
        '(--all -a)'{--all,-a}'[apply to all containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_pause() {
    _arguments \
        '(--all -a)'{--all,-a}'[apply to all containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_registry() {
    local state
    _arguments -C \
        '(--all -a)'{--all,-a}'[show all images including disabled]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1: :_otter_registry_subcommands' \
        '*:: :->reg_sub' \
        && return

    case "${state}" in
        reg_sub)
            case "${words[1]}" in
                pull|get)
                    _arguments \
                        '(--all -a)'{--all,-a}'[pull all images]' \
                        '(--force -f)'{--force,-f}'[force re-pull]' \
                        '(--root -r)'{--root,-r}'[run as root]' \
                        '(--help -h)'{--help,-h}'[show help]'
                    ;;
                remove|rm)
                    _arguments \
                        '(--all -a)'{--all,-a}'[remove all images]' \
                        '(--force -f)'{--force,-f}'[force remove]' \
                        '(--root -r)'{--root,-r}'[run as root]' \
                        '(--help -h)'{--help,-h}'[show help]'
                    ;;
            esac
            ;;
    esac
}

_otter_registry_subcommands() {
    local -a subcmds
    subcmds=(
        'pull:pull an image from the registry (alias: get)'
        'remove:remove a local image (alias: rm)'
    )
    _describe 'registry subcommand' subcmds
}

_otter_remove() {
    _arguments \
        '(--all -a)'{--all,-a}'[remove all containers]' \
        '(--force -f)'{--force,-f}'[force remove]' \
        '(--rm-home -H)'{--rm-home,-H}'[also remove container home directory]' \
        '(--bypass-lock -B)'{--bypass-lock,-B}'[remove even if locked]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_restart() {
    _arguments \
        '(--all -a)'{--all,-a}'[restart all containers]' \
        '(--force -f)'{--force,-f}'[force stop before restart]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_start() {
    _arguments \
        '(--all -a)'{--all,-a}'[start all containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_stop() {
    _arguments \
        '(--all -a)'{--all,-a}'[stop all containers]' \
        '(--force -f)'{--force,-f}'[force stop]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_unlock() {
    _arguments \
        '(--all -a)'{--all,-a}'[apply to all containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter_upgrade() {
    _arguments \
        '(--all -a)'{--all,-a}'[upgrade all containers]' \
        '(--running -R)'{--running,-R}'[upgrade only running containers]' \
        '(--root -r)'{--root,-r}'[run as root]' \
        '(--help -h)'{--help,-h}'[show help]' \
        '1:container:_otter_container_names'
}

_otter "$@"