# otter-list

## Name

`otter list` — list all otter-managed containers.

## Synopsis

```text
otter ls   [options]
otter list [options]
```

`ls` is a full alias of `list`; the two behave identically.

## Description

Lists every container that otter created and manages, along with its
status and image. By default only rootless containers are shown.

## Options

| Flag     | Alias | Description                                                  |
| -------- | ----- | ------------------------------------------------------------ |
| `--root` | `-r`  | List rootful containers instead of rootless ones.            |
| `--json` | `-j`  | Print the list as a JSON array instead of a formatted table. |

## Examples

```sh
otter ls --root
```

Lists rootful containers.

```sh
NO_COLOR=1 otter ls
```

Lists rootless containers with ANSI colors disabled.

```sh
NO_COLOR=1 otter list --root
```

Lists rootful containers with colors disabled.

## Notes

- Rootful containers are hidden by default — pass `--root` to see them.
- `--root` is preferred over running `sudo otter list`. If you need a
  privilege-escalation program other than `sudo`, set
  `OTR_SUDO_PROGRAM` or the `preferences.sudo-program` value in
  `otter.conf`.
- Setting the `NO_COLOR` environment variable to any non-empty value
  disables colored/styled output.

## See Also

- [otter-inspect](otter-inspect.md) — detailed info for a single
  container.
- [otter-interactive](otter-interactive.md) — a TUI alternative to
  `list`.
