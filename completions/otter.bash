# bash completion for otter
# Source this file or drop it in /etc/bash_completion.d/otter
# or ~/.local/share/bash-completion/completions/otter

_otter_containers() {
    otter list --json 2>/dev/null | grep '"name"' | sed 's/.*"name": *"\([^"]*\)".*/\1/'
}

_otter_registry_images() {
    otter registry 2>/dev/null | awk 'NR>1 {print $1}'
}

_otter() {
    local cur prev words cword
    _init_completion -n "=:" || return

    local global_flags="--root --help --version"
    local commands="assemble create enter generate-entry inspect journal list lock pause registry remove restart start stop unlock upgrade"

    local subcommand=""
    local i
    # shellcheck disable=SC2249
    for (( i=1; i < cword; i++ )); do
        case "${words[i]}" in
            assemble|dmf)        subcommand="assemble";       break ;;
            create|mk)           subcommand="create";         break ;;
            enter|sh)            subcommand="enter";          break ;;
            generate-entry|pin)  subcommand="generate-entry"; break ;;
            inspect|info)        subcommand="inspect";        break ;;
            journal|logs)        subcommand="journal";        break ;;
            list|ls)             subcommand="list";           break ;;
            lock|lk)             subcommand="lock";           break ;;
            pause|zz)            subcommand="pause";          break ;;
            registry|reg)        subcommand="registry";       break ;;
            remove|rm)           subcommand="remove";         break ;;
            restart|rbt)         subcommand="restart";        break ;;
            start|up)            subcommand="start";          break ;;
            stop|dn)             subcommand="stop";           break ;;
            unlock|ulk)          subcommand="unlock";         break ;;
            upgrade|syu)         subcommand="upgrade";        break ;;
        esac
    done

    case "${subcommand}" in

        assemble)
            local assemble_sub=""
            # shellcheck disable=SC2249
            for (( i=1; i < cword; i++ )); do
                case "${words[i]}" in
                    create|mk) assemble_sub="create"; break ;;
                    remove|rm) assemble_sub="remove"; break ;;
                esac
            done
            case "${assemble_sub}" in
                create)
                    # shellcheck disable=SC2249
                    case "${prev}" in
                        --file) _filedir '*.toml'; return ;;
                    esac
                    mapfile -t COMPREPLY < <(compgen -W "--file --replace --help" -- "${cur}")
                    ;;
                remove)
                    # shellcheck disable=SC2249
                    case "${prev}" in
                        --file) _filedir '*.toml'; return ;;
                    esac
                    mapfile -t COMPREPLY < <(compgen -W "--file --help" -- "${cur}")
                    ;;
                *)
                    mapfile -t COMPREPLY < <(compgen -W "create remove --help" -- "${cur}")
                    ;;
            esac
            ;;

        create)
            # shellcheck disable=SC2249
            case "${prev}" in
                --image)
                    local images
                    images=$(_otter_registry_images)
                    mapfile -t COMPREPLY < <(compgen -W "${images}" -- "${cur}")
                    return
                    ;;
                --shell)
                    mapfile -t COMPREPLY < <(compgen -W "bash zsh fish" -- "${cur}")
                    return
                    ;;
                --platform)
                    mapfile -t COMPREPLY < <(compgen -W "linux/amd64 linux/arm64 linux/arm/v7 linux/386" -- "${cur}")
                    return
                    ;;
                --clone)
                    local containers
                    containers=$(_otter_containers)
                    mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                    return
                    ;;
                --home|--volume|--additional-flags|--additional-packages| \
                --init-hooks|--pre-init-hooks|--memory|--cpu-threads|--hostname)
                    return
                    ;;
            esac
            mapfile -t COMPREPLY < <(compgen -W "
                --image
                --hostname
                --shell
                --pull
                --clone
                --home
                --volume
                --additional-flags
                --additional-packages
                --init-hooks
                --pre-init-hooks
                --init
                --memory
                --cpu-threads
                --nvidia
                --platform
                --unshare-devsys
                --unshare-groups
                --unshare-ipc
                --unshare-netns
                --unshare-process
                --unshare-all
                --no-entry
                --root
                --help
            " -- "${cur}")
            ;;

        enter)
            # shellcheck disable=SC2249
            case "${prev}" in
                --additional-flags|--add-env) return ;;
            esac
            local has_container=0
            for (( i=1; i < cword; i++ )); do
                [[ "${words[i]}" != -* ]] && has_container=1 && break
            done
            if [[ "${cur}" != -* && "${has_container}" -eq 0 ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "
                --clean-path
                --additional-flags
                --no-tty
                --no-workdir
                --add-env
                --empty-env
                --auto-start
                --root
                --help
            " -- "${cur}")
            ;;

        generate-entry)
            # shellcheck disable=SC2249
            case "${prev}" in
                --icon) return ;;
            esac
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--delete --icon --all --root --help" -- "${cur}")
            ;;

        inspect)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--json --root --help" -- "${cur}")
            ;;

        journal)
            # shellcheck disable=SC2249
            case "${prev}" in
                --since|--until|--tail) return ;;
            esac
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "
                --follow
                --since
                --until
                --timestamps
                --tail
                --root
                --help
            " -- "${cur}")
            ;;

        list)
            mapfile -t COMPREPLY < <(compgen -W "--json --root --help" -- "${cur}")
            ;;

        lock)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --root --help" -- "${cur}")
            ;;

        pause)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --root --help" -- "${cur}")
            ;;

        registry)
            local reg_sub=""
            # shellcheck disable=SC2249
            for (( i=1; i < cword; i++ )); do
                case "${words[i]}" in
                    pull|get)  reg_sub="pull";   break ;;
                    remove|rm) reg_sub="remove"; break ;;
                esac
            done
            case "${reg_sub}" in
                pull)
                    mapfile -t COMPREPLY < <(compgen -W "--all --force --root --help" -- "${cur}")
                    ;;
                remove)
                    mapfile -t COMPREPLY < <(compgen -W "--all --force --root --help" -- "${cur}")
                    ;;
                *)
                    mapfile -t COMPREPLY < <(compgen -W "pull remove --all --help" -- "${cur}")
                    ;;
            esac
            ;;

        remove)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "
                --all
                --force
                --rm-home
                --bypass-lock
                --root
                --help
            " -- "${cur}")
            ;;

        restart)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --force --root --help" -- "${cur}")
            ;;

        start)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --root --help" -- "${cur}")
            ;;

        stop)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --force --root --help" -- "${cur}")
            ;;

        unlock)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --root --help" -- "${cur}")
            ;;

        upgrade)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all --running --root --help" -- "${cur}")
            ;;

        *)
            mapfile -t COMPREPLY < <(compgen -W "${commands} ${global_flags}" -- "${cur}")
            ;;
    esac
}

complete -F _otter otter