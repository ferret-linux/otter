# Otter

Otter is a host-integrated container environment manager. It creates and
manages containerized Linux environments with shared host resources and
optional features such as init systems, exported applications, and
namespace controls. It supports Podman, Docker, and nerdctl.

The repository is organized into a few related parts:

| Path | What it contains |
| --- | --- |
| [`otter/`](otter/) | The Go command-line application, its embedded user documentation, and the build files. |
| [`images/`](images/) | Containerfiles and the catalogs used to build and publish Otter's curated images. |
| [`otter/tests/`](otter/tests/) | End-to-end shell tests for the CLI and supported container managers. |
| [`holt/`](holt/) | The Holt desktop interface for Otter. |
| [`.github/workflows/`](.github/workflows/) | CI, image build, and verification workflows. |

## Start using Otter

Otter is not yet published through a package repository; building from source
is the supported installation path at this time. You need Go 1.27.3 or newer,
GNU Make, and a working Podman, Docker, or nerdctl installation.

```sh
git clone https://github.com/ferret-linux/otter.git
cd otter/otter
make build
./bin/otter docs
./bin/otter create my-fedora --image fedora
./bin/otter enter my-fedora
```

See the [Otter README](otter/README.md) for a fuller overview and quick start.

## Repository guides

- [Otter CLI: installation, usage, configuration, and built-in docs](otter/README.md)
- [Curated images: catalogs, Containerfiles, and build workflow](images/README.md)
- [Testing: Go checks, end-to-end scripts, and CI matrix](otter/tests/README.md)
- [Holt desktop app: development setup and scripts](holt/README.md)

The CLI documentation is embedded in the binary and can be opened with
`otter docs`. Its Markdown source is in
[`otter/pkg/commands/assets/docs/`](otter/pkg/commands/assets/docs/).

## License

Otter is licensed under GPL-3.0-only. See [`otter/LICENSE.md`](otter/LICENSE.md)
and [`THIRD_PARTY_LICENSE.md`](otter/THIRD_PARTY_LICENSE.md) for license
details and third-party notices.
