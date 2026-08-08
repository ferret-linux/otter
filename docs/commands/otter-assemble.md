# otter-assemble

## Name

`otter assemble` — manage otter containers declared in a manifest file.

## Synopsis

```text
otter dmf      <create|remove> [options]
otter assemble <create|remove> [options]
```

`dmf` is a full alias of `assemble`. `create`'s alias is `mk`, `remove`'s
alias is `rm`.

## Description

Reads a TOML manifest describing one or more `[[container]]` entries and
creates or removes the corresponding otter containers in one batch. This
is the declarative counterpart to running individual `otter create`
commands by hand — see [manifests.md](../manifests.md) for the full
manifest schema.

`assemble create` provisions each container defined in the manifest
(equivalent to `otter create` with the manifest's fields mapped onto the
same options), then — if the manifest requests it — starts the container
and runs its init hooks, exports the requested apps/binaries via
`otter-export`, and locks the container. If setup fails partway through,
the just-created container is automatically rolled back (removed) rather
than left in a half-configured state.

`assemble remove` force-removes the named container (or every container
in the manifest, if no name is given).

`--replace` on `create` deletes any existing container with a matching
name before recreating it from the manifest.

## Options

### create / mk

| Flag        | Alias | Description                                                                        |
| ----------- | ----- | ---------------------------------------------------------------------------------- |
| `--file`    | `-f`  | Path or URL to the manifest file. **Required.**                                    |
| `--replace` | `-R`  | Replace an existing container with a matching name instead of leaving it untouched.|

### remove / rm

| Flag     | Alias | Description                                       |
| -------- | ----- | --------------------------------------------------|
| `--file` | `-f`  | Path or URL to the manifest file. **Required.**   |

## Examples

```sh
otter dmf rm my-box --file /path/to/file.ini
```

Removes the `my-box` entry defined in the manifest.

```sh
otter dmf mk --replace --file /path/to/file.ini
```

Creates every container in the manifest, replacing any that already exist.

```sh
otter assemble create -R --file /path/to/file.ini
```

Same as above, using the non-aliased command and short flag.

```sh
otter assemble remove my-box --file /path/to/file.ini
```

Removes the `my-box` entry defined in the manifest, using the non-aliased
command name.

## Notes

- `--file` is required for both subcommands. The positional argument,
  when given, targets a single named container from the manifest — it is
  not the manifest path.
- `--file` accepts either a local filesystem path or an HTTP(S) URL; a
  URL is fetched directly.
- If any manifest entry sets `settings.rootful = true`, otter validates
  that a privilege-escalation program is usable before doing anything,
  even if other entries in the same manifest are rootless.
- Manifest entries can `include` another entry in the same file to
  inherit its settings, with the including entry's own explicit values
  taking priority. See [manifests.md](../manifests.md).

## See Also

- [manifests.md](../manifests.md) — full manifest file schema and merge
  semantics.
- [otter-create](otter-create.md) — the underlying per-container creation
  logic.
