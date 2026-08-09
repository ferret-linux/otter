# otter

`otter` is the command-line tool for creating and managing
host-integrated Linux containers — see [getting-started.md](../getting-started.md)
for an introduction.

Two distinct programs share the name `otter`: the one you install and
run on your **host**, and a small, separate one that otter provisions
automatically **inside every container** for tasks that only make
sense from in there. They're covered separately below.

## On the host

This is the `otter` you install and run yourself. Every command has a
short alias alongside its full name; both behave identically.

| Command                                     | Alias | What it does                                          |
| ------------------------------------------- | ----- | ----------------------------------------------------- |
| [`assemble`](otter-assemble.md)             | `dmf` | Manage containers declared in a manifest file.        |
| [`create`](otter-create.md)                 | `mk`  | Create a native-distro-like container environment.    |
| [`documentation`](otter-documentation.md)   | `docs`| Browse otter's documentation in a terminal viewer.    |
| [`enter`](otter-enter.md)                   | `sh`  | Enter a container environment.                        |
| [`generate-entry`](otter-generate-entry.md) | `pin` | Generate (or delete) a desktop entry for a container. |
| [`inspect`](otter-inspect.md)               | `info`| Show detailed info about a container.                 |
| [`journal`](otter-journal.md)               | `logs`| Show logs of a container.                             |
| [`list`](otter-list.md)                     | `ls`  | List all otter-managed containers.                    |
| [`lock`](otter-lock.md)                     | `lk`  | Lock a container to prevent removal or upgrades.      |
| [`pause`](otter-pause.md)                   | `zz`  | Pause a running container.                            |
| [`registry`](otter-registry.md)             | `reg` | Browse and manage otter's prebuilt image registry.    |
| [`remove`](otter-remove.md)                 | `rm`  | Remove a container environment.                       |
| [`restart`](otter-restart.md)               | `rbt` | Restart a container environment.                      |
| [`start`](otter-start.md)                   | `up`  | Start one or more containers.                         |
| [`stop`](otter-stop.md)                     | `dn`  | Stop one or more running containers.                  |
| [`unlock`](otter-unlock.md)                 | `ulk` | Unlock a container to allow removal and upgrades.     |
| [`upgrade`](otter-upgrade.md)               | `syu` | Upgrade packages inside containers.                   |

Every command above also accepts `--help`/`-h` for its own usage, and
most accept `--root`/`-r` to operate on rootful containers instead of
rootless ones — see each command's own page for its full flag set.

## Inside a container

Every container `otter create` makes also gets its own copy of
`otter`, mounted read-only at `/usr/bin/otter` **inside** the
container. This is a different, much smaller program from the host
CLI above — it only understands two subcommands, and exists purely to
keep the same `otter <verb>` naming convention available from inside a
container:

| Command                                 | What it does                                            |
| --------------------------------------- | ------------------------------------------------------- |
| [`export`](otter-export.md)             | Export an app or binary from the container to the host. |
| [`host-exec`](otter-host-exec.md)       | Run a command on the host from inside the container.    |

To use these, `otter enter` a container first, then run `otter export
...` / `otter host-exec ...` from the shell you land in — see each
command's own page (both start with a "Where this runs" note
explaining the host-vs-container distinction in more detail).

## See Also

- [getting-started.md](../getting-started.md) — installing and taking
  your first steps with otter.
- [configuration.md](../configuration.md) — `otter.conf` and its
  defaults.
- [host-integration.md](../host-integration.md) — how otter integrates
  a container with the host (exports, desktop entries, and more).
