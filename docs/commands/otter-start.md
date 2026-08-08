# otter-start

## Name

`otter start` — start one or more otter-managed container environments.

## Synopsis

```
otter up    [options] [container...]
otter start [options] [container...]
```

`up` is a full alias of `start`.

## Description

Starts the named container(s), or every otter-managed container with
`--all`. Multiple names can be given as a single comma-separated argument
(e.g. `otter start foo,bar`). Containers that are already running are
skipped without being treated as a failure.

Every requested name is always attempted, even if starting an earlier one
fails; a final summary line reports how many succeeded, failed, or were
skipped.

## Options

| Flag | Alias | Description |
|---|---|---|
| `--all` | `-a` | Start every otter container instead of specific names. |
| `--root` | `-r` | Operate on rootful containers. |

## Examples

```
otter up --all
```
Starts every rootless otter container.

```
otter up my-box
```
Starts `my-box`.

```
otter start my-box
```
Same as above, using the non-aliased command name.

```
otter start --root my-box
```
Starts the rootful container `my-box`.

## Notes

- Already-running containers are skipped, not treated as an error.
- You must supply either at least one container name or `--all`.

## See Also

- [otter-stop](otter-stop.md), [otter-restart](otter-restart.md), [otter-pause](otter-pause.md)