# Otter CLI

Otter creates and manages host-integrated Linux container environments. It
works with Podman, Docker, or nerdctl and includes commands for creating,
entering, starting, inspecting, and removing environments. Containers can
also use optional init systems, resource limits, GPU access, and exported
host applications.

Otter is not yet published through a package repository. Building from source
is the currently supported installation path.

## Requirements

- Go 1.27.3 or newer
- GNU Make
- A working installation of Podman, Docker, or nerdctl

## Build and try it

From the repository root:

```sh
cd otter
make build
./bin/otter docs
./bin/otter create my-fedora --image fedora
./bin/otter enter my-fedora
./bin/otter remove my-fedora
```

The build creates `./bin/otter`. `make install` installs the binary and shell
completions; see the [installation guide](pkg/commands/assets/docs/01.Installation.md)
for install options.

## Documentation and help

The CLI includes an offline documentation browser:

```sh
otter docs
```

The documentation is embedded into the binary, so it matches the version you
are running. The source Markdown is in
[`pkg/commands/assets/docs/`](pkg/commands/assets/docs/). Start with the
[Getting Started guide](pkg/commands/assets/docs/02.Getting%20Started.md), or
use `otter --help` for a command summary.

## Further reading

- [Installation](pkg/commands/assets/docs/01.Installation.md)
- [Command reference](pkg/commands/assets/docs/03.Commands/)
- [Concepts](pkg/commands/assets/docs/04.Concepts/)
- [Tutorials](pkg/commands/assets/docs/05.Tutorials/)
- [FAQ and troubleshooting](pkg/commands/assets/docs/06.FAQ%20%26%20Troubleshooting.md)
- [Caveats and considerations](pkg/commands/assets/docs/07.Caveats%20%26%20Considerations.md)
- [Choosing a container tool](pkg/commands/assets/docs/08.Choosing%20Your%20Tool.md)
- [Image build and catalog guide](../images/README.md)
- [Test guide](tests/README.md)

## License and credits

Otter is licensed under GPL-3.0-only. See [`LICENSE.md`](LICENSE.md) and
[`THIRD_PARTY_LICENSE.md`](THIRD_PARTY_LICENSE.md). Portions of the project
build on [Distrobox](https://github.com/89luca89/distrobox); thanks to Luca Di
Maio and the Distrobox contributors.
