# otter-pause

## Name

`otter pause` — pause a running otter-managed container.

## Synopsis

```text
otter zz    [options] [container...]
otter pause [options] [container...]
```

`zz` is a full alias of `pause`.

## Description

Freezes the named container(s) in memory using the container manager's
native pause/freeze support, or every currently-running otter container
with `--all`. Containers that are not currently running are
automatically skipped rather than treated as an error — this includes
the `--all` case, where only the subset of running containers is
targeted.

## Options

| Flag     | Alias | Description                                    |
| -------- | ----- | ---------------------------------------------- |
| `--all`  | `-a`  | Pause every currently-running otter container. |
| `--root` | `-r`  | Pause rootful containers.                      |

## Examples

```sh
otter zz my-box
```

Pauses `my-box`.

```sh
otter zz --all
```

Pauses every running rootless otter container.

```sh
otter pause ubuntu,fedora
```

Pauses both `ubuntu` and `fedora` in one call (comma-separated names).

```sh
otter pause --all --root
```

Pauses every running rootful otter container.

## Notes

- Paused containers are frozen in memory. Use `otter start` to resume a
  paused container (there is no separate "unpause" command).
- Rootful containers need `--root` to be found and acted on.

## See Also

- [otter-start](otter-start.md) — resumes a paused container.
- [otter-stop](otter-stop.md), [otter-restart](otter-restart.md)
