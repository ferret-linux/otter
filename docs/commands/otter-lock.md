# otter-lock

`otter lock` — lock a container to prevent removal or upgrades.

## Synopsis

```text
otter lk   [options] [container...]
otter lock [options] [container...]
```

`lk` is a full alias of `lock`.

## Description

Marks the named container(s) — or every otter container with `--all` —
as locked, by writing a lock file to `/usr/lib/otter/container.lock`
inside each container. A locked container is skipped by `otter remove`
(unless `--bypass-lock` is used) and by `otter upgrade`. The container
must be running for `lock` to write the file into it. Already-locked
containers are skipped without being treated as a failure.

## Options

| Flag     | Alias | Description                 |
| -------- | ----- | --------------------------- |
| `--all`  | `-a`  | Lock every otter container. |
| `--root` | `-r`  | Lock a rootful container.   |

## Examples

```sh
otter lk my-box
```

Locks `my-box`.

```sh
otter lk my-box --root
```

Locks the rootful container `my-box`.

```sh
otter lock my-box
```

Same as the first example, using the non-aliased command name.

## Notes

- Locked containers are skipped by `remove` and `upgrade`. Use
  `otter unlock` to reverse.
- The container must be running to lock — start it first with
  `otter start` if needed.
- Already-locked containers are skipped, not treated as an error.
- Lock state lives inside the container filesystem at
  `/usr/lib/otter/container.lock`, so it travels with the container
  (e.g. survives `otter restart`) but is lost if the container is
  removed and recreated.

## See Also

- [otter-unlock](otter-unlock.md)
- [otter-remove](otter-remove.md), [otter-upgrade](otter-upgrade.md) —
  both respect lock state.
