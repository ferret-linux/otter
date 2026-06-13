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

    local global_flags="--root -r --help -h --version"
    local commands="assemble create enter generate-entry inspect journal list lock pause registry remove restart start stop unlock upgrade"

    # Depth-1: what is the subcommand?
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
                        --file|-f) _filedir '*.toml'; return ;;
                    esac
                    mapfile -t COMPREPLY < <(compgen -W "--file -f --replace -R --help -h" -- "${cur}")
                    ;;
                remove)
                    # shellcheck disable=SC2249
                    case "${prev}" in
                        --file|-f) _filedir '*.toml'; return ;;
                    esac
                    mapfile -t COMPREPLY < <(compgen -W "--file -f --help -h" -- "${cur}")
                    ;;
                *)
                    mapfile -t COMPREPLY < <(compgen -W "create remove --help -h" -- "${cur}")
                    ;;
            esac
            ;;

        create)
            # shellcheck disable=SC2249
            case "${prev}" in
                --image|-i)
                    local images
                    images=$(_otter_registry_images)
                    mapfile -t COMPREPLY < <(compgen -W "${images}" -- "${cur}")
                    return
                    ;;
                --shell|-s)
                    mapfile -t COMPREPLY < <(compgen -W "bash zsh fish" -- "${cur}")
                    return
                    ;;
                --platform|-P)
                    mapfile -t COMPREPLY < <(compgen -W "linux/amd64 linux/arm64 linux/arm/v7 linux/386" -- "${cur}")
                    return
                    ;;
                --clone|-C)
                    local containers
                    containers=$(_otter_containers)
                    mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                    return
                    ;;
                --home|-H|--volume|-v|--additional-flags|-a|--additional-packages|-ap| \
                --init-hooks|-ih|--pre-init-hooks|-ph|--memory|-m|--cpu-threads|-t|--hostname|-n)
                    return
                    ;;
            esac
            mapfile -t COMPREPLY < <(compgen -W "
                --image -i
                --hostname -n
                --shell -s
                --pull -p
                --clone -C
                --home -H
                --volume -v
                --additional-flags -a
                --additional-packages -ap
                --init-hooks -ih
                --pre-init-hooks -ph
                --init -I
                --memory -m
                --cpu-threads -t
                --nvidia -N
                --platform -P
                --unshare-devsys -ud
                --unshare-groups -ug
                --unshare-ipc -ui
                --unshare-netns -un
                --unshare-process -up
                --unshare-all -ua
                --no-entry -E
                --root -r
                --help -h
            " -- "${cur}")
            ;;

        enter)
            # shellcheck disable=SC2249
            case "${prev}" in
                --additional-flags|-a|--add-env|-e) return ;;
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
                --clean-path -c
                --additional-flags -a
                --no-tty -T
                --no-workdir -nw
                --add-env -e
                --empty-env -E
                --auto-start -S
                --root -r
                --help -h
            " -- "${cur}")
            ;;

        generate-entry)
            # shellcheck disable=SC2249
            case "${prev}" in
                --icon|-i) return ;;
            esac
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--delete -d --icon -i --all -a --root -r --help -h" -- "${cur}")
            ;;

        inspect)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--json -j --root -r --help -h" -- "${cur}")
            ;;

        journal)
            # shellcheck disable=SC2249
            case "${prev}" in
                --since|-s|--until|-u|--tail|-n) return ;;
            esac
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "
                --follow -f
                --since -s
                --until -u
                --timestamps -t
                --tail -n
                --root -r
                --help -h
            " -- "${cur}")
            ;;

        list)
            mapfile -t COMPREPLY < <(compgen -W "--json -j --root -r --help -h" -- "${cur}")
            ;;

        lock)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --root -r --help -h" -- "${cur}")
            ;;

        pause)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --root -r --help -h" -- "${cur}")
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
                    mapfile -t COMPREPLY < <(compgen -W "--all -a --force -f --root -r --help -h" -- "${cur}")
                    ;;
                remove)
                    mapfile -t COMPREPLY < <(compgen -W "--all -a --force -f --root -r --help -h" -- "${cur}")
                    ;;
                *)
                    mapfile -t COMPREPLY < <(compgen -W "pull remove --all -a --help -h" -- "${cur}")
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
                --all -a
                --force -f
                --rm-home -H
                --bypass-lock -B
                --root -r
                --help -h
            " -- "${cur}")
            ;;

        restart)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --force -f --root -r --help -h" -- "${cur}")
            ;;

        start)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --root -r --help -h" -- "${cur}")
            ;;

        stop)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --force -f --root -r --help -h" -- "${cur}")
            ;;

        unlock)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --root -r --help -h" -- "${cur}")
            ;;

        upgrade)
            if [[ "${cur}" != -* ]]; then
                local containers
                containers=$(_otter_containers)
                mapfile -t COMPREPLY < <(compgen -W "${containers}" -- "${cur}")
                return
            fi
            mapfile -t COMPREPLY < <(compgen -W "--all -a --running -R --root -r --help -h" -- "${cur}")
            ;;

        *)
            mapfile -t COMPREPLY < <(compgen -W "${commands} ${global_flags}" -- "${cur}")
            ;;
    esac
}

complete -F _otter otter