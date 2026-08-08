# otter-restart

## Name

`otter restart` — restart an otter-managed container environment.

## Synopsis

```
otter rbt     [options] [container...]
otter restart [options] [container...]
```

`rbt` is a full alias of `restart`.

## Description

Restarts the named container(s), or every otter-managed container with
`--all`. Internally this simply calls stop followed by start for each
container — there is no separate "restart" primitive in the underlying
container manager. Because of this, `restart` works on containers in any
state: on an already-stopped container it effectively just starts it.

## Options

| Flag | Alias | Description |
|---|---|---|
| `--all` | `-a` | Restart every otter container instead of specific names. |
| `--root` | `-r` | Restart rootful containers. |
| `--force` | `-f` | Force the stop phase instead of a graceful stop. |

## Examples

```
otter rbt my-box
```
Restarts `my-box`.

```
otter rbt --all
```
Restarts every rootless otter container.

```
otter restart --all --root
```
Restarts every rootful otter container.

## Notes

- Wraps `otter stop` and `otter start` internally.
- Works on both running and stopped containers — on a stopped container it
  simply starts it.

## See Also

- [otter-start](otter-start.md), [otter-stop](otter-stop.md)