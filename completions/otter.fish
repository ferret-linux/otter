# fish completion for otter
# Drop in ~/.config/fish/completions/otter.fish

# Disable file completion by default
complete -c otter -f

# Helper: list container names
function __otter_containers
    otter list --json 2>/dev/null | string match -r '"name": *"[^"]*"' | string replace -r '.*"name": *"([^"]*)".*' '$1'
end

# Helper: list registry image names
function __otter_registry_images
    otter registry 2>/dev/null | tail -n +2 | awk '{print $1}'
end

# Helper: true when no subcommand has been given yet
function __otter_no_subcommand
    not __fish_seen_subcommand_from \
        assemble dmf \
        create mk \
        enter sh \
        generate-entry pin \
        inspect info \
        journal logs \
        list ls \
        lock lk \
        pause zz \
        registry reg \
        remove rm \
        restart rbt \
        start up \
        stop dn \
        unlock ulk \
        upgrade syu
end

function __otter_using_subcommand
    __fish_seen_subcommand_from $argv
end

# ── Global flags ──────────────────────────────────────────────────────────────
complete -c otter -n __otter_no_subcommand -l root    -s r -d 'Run as root'
complete -c otter -n __otter_no_subcommand -l help    -s h -d 'Show help'
complete -c otter -n __otter_no_subcommand -l version       -d 'Show version'

# ── Top-level subcommands ─────────────────────────────────────────────────────
complete -c otter -n __otter_no_subcommand -a assemble       -d 'Apply a manifest file'
complete -c otter -n __otter_no_subcommand -a dmf            -d 'Apply a manifest file (alias: assemble)'
complete -c otter -n __otter_no_subcommand -a create         -d 'Create a new container'
complete -c otter -n __otter_no_subcommand -a mk             -d 'Create a new container (alias: create)'
complete -c otter -n __otter_no_subcommand -a enter          -d 'Enter a container shell'
complete -c otter -n __otter_no_subcommand -a sh             -d 'Enter a container shell (alias: enter)'
complete -c otter -n __otter_no_subcommand -a generate-entry -d 'Create a desktop entry'
complete -c otter -n __otter_no_subcommand -a pin            -d 'Create a desktop entry (alias: generate-entry)'
complete -c otter -n __otter_no_subcommand -a inspect        -d 'Show container details'
complete -c otter -n __otter_no_subcommand -a info           -d 'Show container details (alias: inspect)'
complete -c otter -n __otter_no_subcommand -a journal        -d 'Show container logs'
complete -c otter -n __otter_no_subcommand -a logs           -d 'Show container logs (alias: journal)'
complete -c otter -n __otter_no_subcommand -a list           -d 'List otter containers'
complete -c otter -n __otter_no_subcommand -a ls             -d 'List otter containers (alias: list)'
complete -c otter -n __otter_no_subcommand -a lock           -d 'Lock a container'
complete -c otter -n __otter_no_subcommand -a lk             -d 'Lock a container (alias: lock)'
complete -c otter -n __otter_no_subcommand -a pause          -d 'Pause a container'
complete -c otter -n __otter_no_subcommand -a zz             -d 'Pause a container (alias: pause)'
complete -c otter -n __otter_no_subcommand -a registry       -d 'Manage otter images'
complete -c otter -n __otter_no_subcommand -a reg            -d 'Manage otter images (alias: registry)'
complete -c otter -n __otter_no_subcommand -a remove         -d 'Remove a container'
complete -c otter -n __otter_no_subcommand -a rm             -d 'Remove a container (alias: remove)'
complete -c otter -n __otter_no_subcommand -a restart        -d 'Restart a container'
complete -c otter -n __otter_no_subcommand -a rbt            -d 'Restart a container (alias: restart)'
complete -c otter -n __otter_no_subcommand -a start          -d 'Start a container'
complete -c otter -n __otter_no_subcommand -a up             -d 'Start a container (alias: start)'
complete -c otter -n __otter_no_subcommand -a stop           -d 'Stop a container'
complete -c otter -n __otter_no_subcommand -a dn             -d 'Stop a container (alias: stop)'
complete -c otter -n __otter_no_subcommand -a unlock         -d 'Unlock a container'
complete -c otter -n __otter_no_subcommand -a ulk            -d 'Unlock a container (alias: unlock)'
complete -c otter -n __otter_no_subcommand -a upgrade        -d 'Upgrade packages in a container'
complete -c otter -n __otter_no_subcommand -a syu            -d 'Upgrade packages in a container (alias: upgrade)'

# ── assemble ──────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand assemble dmf; and not __fish_seen_subcommand_from create mk remove rm' \
    -a create -d 'Create containers from manifest'
complete -c otter -n '__otter_using_subcommand assemble dmf; and not __fish_seen_subcommand_from create mk remove rm' \
    -a mk     -d 'Create containers from manifest (alias: create)'
complete -c otter -n '__otter_using_subcommand assemble dmf; and not __fish_seen_subcommand_from create mk remove rm' \
    -a remove -d 'Remove containers from manifest'
complete -c otter -n '__otter_using_subcommand assemble dmf; and not __fish_seen_subcommand_from create mk remove rm' \
    -a rm     -d 'Remove containers from manifest (alias: remove)'

complete -c otter -n '__otter_using_subcommand assemble dmf; and __fish_seen_subcommand_from create mk' \
    -l file    -s f -d 'Manifest file' -r -F
complete -c otter -n '__otter_using_subcommand assemble dmf; and __fish_seen_subcommand_from create mk' \
    -l replace -s R -d 'Replace existing containers'
complete -c otter -n '__otter_using_subcommand assemble dmf; and __fish_seen_subcommand_from remove rm' \
    -l file    -s f -d 'Manifest file' -r -F
complete -c otter -n '__otter_using_subcommand assemble dmf' \
    -l help    -s h -d 'Show help'

# ── create ────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand create mk' \
    -l image             -s i  -d 'Container image'             -r -a '(__otter_registry_images)'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l hostname          -s n  -d 'Hostname inside container'   -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l shell             -s s  -d 'Shell to use'                -r -a 'bash zsh fish'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l pull              -s p  -d 'Always pull image before creating'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l clone             -s C  -d 'Clone from existing container' -r -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l home              -s H  -d 'Custom home directory'       -r -F
complete -c otter -n '__otter_using_subcommand create mk' \
    -l volume            -s v  -d 'Additional volume mount'     -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l additional-flags  -s a  -d 'Extra flags for container runtime' -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l additional-packages -s ap -d 'Packages to install'      -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l init-hooks        -s ih -d 'Post-init hook commands'     -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l pre-init-hooks    -s ph -d 'Pre-init hook commands'      -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l init              -s I  -d 'Enable init system (systemd)'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l memory            -s m  -d 'Memory limit (e.g. 512m, 2g)' -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l cpu-threads       -s t  -d 'CPU thread limit'            -r
complete -c otter -n '__otter_using_subcommand create mk' \
    -l nvidia            -s N  -d 'Enable nvidia GPU support'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l platform          -s P  -d 'Container platform'          -r -a 'linux/amd64 linux/arm64 linux/arm/v7 linux/386'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-devsys    -s ud -d 'Unshare host devices and sysfs'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-groups    -s ug -d 'Unshare host groups'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-ipc       -s ui -d 'Unshare host IPC namespace'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-netns     -s un -d 'Unshare host network namespace'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-process   -s up -d 'Unshare host process namespace'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l unshare-all       -s ua -d 'Unshare all namespaces'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l no-entry          -s E  -d 'Skip desktop entry generation'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l root              -s r  -d 'Run as root'
complete -c otter -n '__otter_using_subcommand create mk' \
    -l help              -s h  -d 'Show help'

# ── enter ─────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand enter sh' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l clean-path        -s c  -d 'Use clean standard PATH'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l additional-flags  -s a  -d 'Extra flags for exec'        -r
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l no-tty            -s T  -d 'Disable TTY allocation'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l no-workdir        -s nw -d 'Use home dir instead of cwd'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l add-env           -s e  -d 'Pass additional env vars'    -r
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l empty-env         -s E  -d 'Start with empty environment'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l auto-start        -s S  -d 'Start container if stopped'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l root              -s r  -d 'Run as root'
complete -c otter -n '__otter_using_subcommand enter sh' \
    -l help              -s h  -d 'Show help'

# ── generate-entry ────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -l delete -s d -d 'Remove desktop entry'
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -l icon   -s i -d 'Icon path or "auto"'                     -r -F
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -l all    -s a -d 'Apply to all containers'
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -l root   -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand generate-entry pin' \
    -l help   -s h -d 'Show help'

# ── inspect ───────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand inspect info' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand inspect info' \
    -l json -s j -d 'Output as JSON'
complete -c otter -n '__otter_using_subcommand inspect info' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand inspect info' \
    -l help -s h -d 'Show help'

# ── journal ───────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand journal logs' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l follow     -s f -d 'Follow log output'
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l since      -s s -d 'Show logs since timestamp'           -r
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l until      -s u -d 'Show logs until timestamp'           -r
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l timestamps -s t -d 'Show timestamps'
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l tail       -s n -d 'Number of lines to show'             -r
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l root       -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand journal logs' \
    -l help       -s h -d 'Show help'

# ── list ──────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand list ls' \
    -l json -s j -d 'Output as JSON'
complete -c otter -n '__otter_using_subcommand list ls' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand list ls' \
    -l help -s h -d 'Show help'

# ── lock ──────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand lock lk' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand lock lk' \
    -l all  -s a -d 'Apply to all containers'
complete -c otter -n '__otter_using_subcommand lock lk' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand lock lk' \
    -l help -s h -d 'Show help'

# ── pause ─────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand pause zz' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand pause zz' \
    -l all  -s a -d 'Pause all containers'
complete -c otter -n '__otter_using_subcommand pause zz' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand pause zz' \
    -l help -s h -d 'Show help'

# ── registry ──────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand registry reg; and not __fish_seen_subcommand_from pull get remove rm' \
    -a pull   -d 'Pull an image from the registry'
complete -c otter -n '__otter_using_subcommand registry reg; and not __fish_seen_subcommand_from pull get remove rm' \
    -a get    -d 'Pull an image from the registry (alias: pull)'
complete -c otter -n '__otter_using_subcommand registry reg; and not __fish_seen_subcommand_from pull get remove rm' \
    -a remove -d 'Remove a local image'
complete -c otter -n '__otter_using_subcommand registry reg; and not __fish_seen_subcommand_from pull get remove rm' \
    -a rm     -d 'Remove a local image (alias: remove)'
complete -c otter -n '__otter_using_subcommand registry reg' \
    -l all  -s a -d 'Show/apply to all images'
complete -c otter -n '__otter_using_subcommand registry reg' \
    -l help -s h -d 'Show help'

complete -c otter -n '__otter_using_subcommand registry reg; and __fish_seen_subcommand_from pull get remove rm' \
    -l force -s f -d 'Force operation'
complete -c otter -n '__otter_using_subcommand registry reg; and __fish_seen_subcommand_from pull get remove rm' \
    -l root  -s r -d 'Run as root'

# ── remove ────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand remove rm' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l all         -s a -d 'Remove all containers'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l force       -s f -d 'Force remove'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l rm-home     -s H -d 'Also remove container home directory'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l bypass-lock -s B -d 'Remove even if locked'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l root        -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand remove rm' \
    -l help        -s h -d 'Show help'

# ── restart ───────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand restart rbt' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand restart rbt' \
    -l all   -s a -d 'Restart all containers'
complete -c otter -n '__otter_using_subcommand restart rbt' \
    -l force -s f -d 'Force stop before restart'
complete -c otter -n '__otter_using_subcommand restart rbt' \
    -l root  -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand restart rbt' \
    -l help  -s h -d 'Show help'

# ── start ─────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand start up' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand start up' \
    -l all  -s a -d 'Start all containers'
complete -c otter -n '__otter_using_subcommand start up' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand start up' \
    -l help -s h -d 'Show help'

# ── stop ──────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand stop dn' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand stop dn' \
    -l all   -s a -d 'Stop all containers'
complete -c otter -n '__otter_using_subcommand stop dn' \
    -l force -s f -d 'Force stop'
complete -c otter -n '__otter_using_subcommand stop dn' \
    -l root  -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand stop dn' \
    -l help  -s h -d 'Show help'

# ── unlock ────────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand unlock ulk' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand unlock ulk' \
    -l all  -s a -d 'Apply to all containers'
complete -c otter -n '__otter_using_subcommand unlock ulk' \
    -l root -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand unlock ulk' \
    -l help -s h -d 'Show help'

# ── upgrade ───────────────────────────────────────────────────────────────────
complete -c otter -n '__otter_using_subcommand upgrade syu' \
    -a '(__otter_containers)'
complete -c otter -n '__otter_using_subcommand upgrade syu' \
    -l all     -s a -d 'Upgrade all containers'
complete -c otter -n '__otter_using_subcommand upgrade syu' \
    -l running -s R -d 'Upgrade only running containers'
complete -c otter -n '__otter_using_subcommand upgrade syu' \
    -l root    -s r -d 'Run as root'
complete -c otter -n '__otter_using_subcommand upgrade syu' \
    -l help    -s h -d 'Show help'