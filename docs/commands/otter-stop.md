# otter-stop

## Name

`otter stop` — stop one or more running otter-managed container
environments.

## Synopsis

```text
otter dn   [options] [container...]
otter stop [options] [container...]
```

`dn` is a full alias of `stop`.

## Description

Stops the named container(s), or every otter-managed container with
`--all`. Multiple names can be given as a single comma-separated
argument. Every requested container is first checked to exist (an
unknown name is reported as a per-item failure rather than aborting
the whole batch), then stopped.

## Options

| Flag      | Alias | Description                                           |
| --------- | ----- | ----------------------- ----------------------------- |
| `--all`   | `-a`  | Stop every otter container instead of specific names. |
| `--root`  | `-r`  | Stop rootful containers.                              |
| `--force` | `-f`  | Force-stop containers instead of a graceful stop.     |

## Examples

```sh
otter dn my-box
```

Stops `my-box`.

```sh
otter dn --all
```

Stops every rootless otter container.

```sh
otter stop --all --root
```

Stops every rootful otter container.

## Notes

- `--root` is preferred over `sudo otter stop`. Set
  `OTR_SUDO_PROGRAM` (or the `preferences.sudo-program` config value)
  to use a different privilege-escalation program.
- Rootful containers need `--root` for `stop` to find and act on them.

## See Also

- [otter-start](otter-start.md), [otter-restart](otter-restart.md),
  [otter-pause](otter-pause.md)
