# Otter tests

This directory contains end-to-end shell tests for the Otter CLI. The Go
module's package checks are run separately from the `otter/` directory.

## Go checks

From `otter/`:

```sh
make test
```

`make test` runs `go vet ./...` followed by `go test -v ./...`. The repository
CI also builds the Go module for amd64 and arm64 and runs Go lint and
vulnerability checks; see [`ci-tests.yml`](../../.github/workflows/ci-tests.yml).

## End-to-end tests

The scripts in `e2e/ci/` exercise common CLI operations against a real
container manager. The current distro scripts are:

| Script | Test image |
| --- | --- |
| `alpine.sh` | Alpine |
| `arch.sh` | Arch Linux |
| `fedora.sh` | Fedora |
| `gentoo.sh` | Gentoo |
| `rocky.sh` | Rocky Linux |
| `tumbleweed.sh` | openSUSE Tumbleweed |
| `ubuntu.sh` | Ubuntu |
| `void.sh` | Void Linux |

Each script accepts a runtime and mode:

```sh
bash otter/tests/e2e/ci/alpine.sh podman rootless
```

Supported runtimes are `podman`, `docker`, and `nerdctl`; modes are `rootless`
and `rootful`. Rootless nerdctl is not supported and is skipped by the
scripts. To run locally, first build Otter and put its binary on `PATH`:

```sh
cd otter
make build
cd ..
export PATH="$PWD/otter/bin:$PATH"
bash otter/tests/e2e/ci/alpine.sh podman rootless
```

The end-to-end scripts pull images and create containers named
`otter-test-<distro>` and `otter-test-<distro>-clone`. Their cleanup handler
force-removes those names when the script exits. Before running one, make sure
you do not have containers with those names that you need to keep. Rootful
mode requires permission to run the selected container manager as root.

## Continuous integration

[`ci-tests.yml`](../../.github/workflows/ci-tests.yml) runs on pushes and pull
requests. It builds the Go application, runs lint and security checks, and
calls the end-to-end workflows on amd64 and arm64. The end-to-end matrix
covers Podman and Docker in rootless and rootful modes, plus nerdctl in
rootful mode.
