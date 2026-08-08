# otter-inspect

## Name

`otter inspect` — show detailed info about an otter-managed container.

## Synopsis

```
otter info    [options] [container]
otter inspect [options] [container]
```

`otter info` is a full alias of `otter inspect`.

## Description

Displays a formatted table (or, with `--json`, structured JSON) of the
container's configuration as it was defined at creation time: general
info (name, ID, created timestamp, status, image, platform, hostname,
shell, home directory, lock state, rootful/rootless, and which container
manager is in use), resource limits (memory, CPU threads), enabled
features (init system, Nvidia), and namespace isolation settings (IPC,
network, process, devices, groups, and the rootless userns size limit).

## Options

| Flag | Alias | Description |
|---|---|---|
| `--root` | `-r` | Inspect a rootful container. |
| `--json` | `-j` | Print the result as a JSON object instead of a formatted table. |

## Examples

```
otter info my-box
```
Shows a formatted info table for `my-box`.

```
otter info my-box --root
```
Shows info for the rootful container `my-box`.

```
otter inspect my-box
```
Same as the first example, using the non-aliased command name.

```
otter inspect my-box --root
```
Same as the second example, using the non-aliased command name.

## Notes

- `inspect` only accepts a single container name — it does not support
  `--all` or comma-separated names.
- The `Home` field reflects the actual host directory in use: for
  containers created with a custom `--home`, this is the custom host
  source path, not the canonical in-container path.
- The isolation fields report whether each namespace is currently
  *shared* with the host or *unshared* — the inverse phrasing of the
  `--unshare-*` flags used at `create` time.

## See Also

- [otter-list](otter-list.md) — a lighter-weight summary across all containers.
- [otter-journal](otter-journal.md) — log output, as distinct from static configuration.