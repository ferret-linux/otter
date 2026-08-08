# otter-interactive

`otter interactive` — launch otter's interactive terminal UI.

## Synopsis

```text
otter tui
otter interactive
```

`otter tui` is a full alias of `otter interactive`.

## Description

Launches a full-screen terminal UI (built with Bubble Tea) that lists
all otter-managed containers with their live status, and shows a detail
panel (container ID, image, status) for whichever container is
currently selected. It's a browsable alternative to `otter list` for
quickly scanning your containers.

## Options

| Flag     | Alias | Description            |
| -------- | ----- | ---------------------- |
| `--help` | `-h`  | Show the help message. |

## Examples

```sh
otter tui
```

Launches the interactive UI.

```sh
otter interactive
```

Same as above, using the non-aliased command name.

## Key bindings

| Key            | Action                       |
| -------------- | ---------------------------- |
| `↑` / `↓`      | Navigate the container list. |
| `q` / `Ctrl+C` | Quit.                        |

## Notes

- Also aliased as `otter tui`, for muscle memory if you're used to
  other tools' `tui` subcommands.
- The status colors and dot markers mirror the conventions used by
  `otter list`'s table output.

## See Also

- [otter-list](otter-list.md) — the non-interactive equivalent.
- [otter-inspect](otter-inspect.md) — full configuration detail for a
  single container.
