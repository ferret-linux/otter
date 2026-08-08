# otter-interactive

`otter interactive` — launch otter's interactive terminal UI.

## Synopsis

```text
otter tui         [options]
otter interactive [options]
```

`tui` is a full alias of `interactive`; the two behave identically.

## Description

Launches a full-screen terminal UI (built with Bubble Tea) for browsing
otter containers, arranged as five tabs across the top: **Home**,
**Shell**, **Registry**, **Create**, and **Docs**. Currently only
**Home** is implemented — it lists all otter-managed containers with
their live status on the left, and an action menu (Enter, Start/Stop,
Restart, Pause, Lock, Upgrade, Logs, Remove) for whichever container is
selected on the right. The other four tabs are reserved for future
functionality and currently render as "coming soon" placeholders.

It's a browsable, visual companion to `otter list`, useful for quickly
scanning your containers without memorizing flags.

## Options

| Flag     | Alias |
| -------- | ----- |
| `--root` | `-r`  |
| `--help` | `-h`  |

## Options Explained

### `--root`, `-r`

Browse rootful containers instead of rootless ones.

### `--help`, `-h`

Show the help message.

## Examples

```sh
otter tui
```

Launches the interactive UI, browsing rootless containers on the Home
tab.

```sh
otter interactive
```

Same as above, using the non-aliased command name.

```sh
otter tui --root
```

Launches the UI browsing rootful containers instead.

```sh
otter interactive --root
```

Same as above, using the non-aliased command name. Once inside, press
`2`–`5` (or `→`) to preview the Shell/Registry/Create/Docs tabs, and `1`
(or `←` back to the start) to return to Home.

## Key bindings

| Key                    | Action                                                          |
| ---------------------- | ----------------------------------------------------------------|
| `→` / `l`              | Switch to the next tab.                                         |
| `←` / `h`              | Switch to the previous tab.                                     |
| `1`–`5`                | Jump directly to Home / Shell / Registry / Create / Docs.       |
| `Tab` *(Home)*         | Switch focus between the container list and the action menu.    |
| `↑` / `↓` / `k` / `j`  | Move within the focused pane (container list or action menu).   |
| `Enter`                | Confirm the highlighted item.                                   |
| `q` / `Ctrl+C`         | Quit.                                                           |

## Notes

- Only the Home tab is currently functional; Shell, Registry, Create,
  and Docs are placeholders reserved for future releases.
- On the Home tab, the action menu is navigable but doesn't yet execute
  actions when you press `Enter` — for now, use the dedicated commands
  (`otter enter`, `otter start`, `otter remove`, etc.) to actually act
  on a container.
- The status colors and dot markers on the container list mirror the
  conventions used by `otter list`'s table output.

## See Also

- [otter-list](otter-list.md) — the non-interactive equivalent for the
  container list.
- [otter-inspect](otter-inspect.md) — full configuration detail for a
  single container.
