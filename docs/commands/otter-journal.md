# otter-journal

## Name

`otter journal` — show logs of an otter-managed container.

## Synopsis

```text
otter logs    [options] [container]
otter journal [options] [container]
```

`otter logs` is a full alias of `otter journal`.

## Description

Shows the container manager's own log output for a single container
(equivalent to `podman logs` / `docker logs` / `nerdctl logs` under the
hood, with otter's flags mapped onto the manager's own log options).

## Options

| Flag           | Alias | Description                             |
| -------------- | ----- | --------------------------------------- |
| `--tail`       | `-n`  | Show only the last N lines.             |
| `--since`      | `-s`  | Show logs generated since a given time. |
| `--until`      | `-u`  | Show logs generated until a given time. |
| `--follow`     | `-f`  | Stream logs live, like `tail -f`.       |
| `--timestamps` | `-t`  | Prefix each line with its timestamp.    |
| `--root`       | `-r`  | View a rootful container's logs.        |

## Examples

```sh
otter logs my-box --follow
```

Streams `my-box`'s logs live.

```sh
otter logs my-box --tail 50 --timestamps
```

Shows the last 50 log lines of `my-box`, with timestamps.

```sh
otter journal my-box
```

Shows all available logs for `my-box`.

```sh
otter journal my-box --since 1h --timestamps
```

Shows the last hour of logs for `my-box`, with timestamps.

## Notes

- `--since`/`--until` accept durations like `30s`, `15m`, `2h`, `1h30m`,
  or absolute timestamps like `2026-05-01T00:00:00`.
- `journal` only accepts a single container name — unlike most other
  otter commands, it does not support `--all` or comma-separated names.

## See Also

- [otter-inspect](otter-inspect.md) — static configuration info, as
  distinct from log output.
