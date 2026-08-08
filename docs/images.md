# Images

Otter maintains a curated registry of prebuilt, otter-optimized
container images — one or more per distro — described by a JSON
manifest (`images-properties.json`) fetched from the upstream
repository at runtime. Browsing, pulling, and removing these is done
with [otter registry](commands/otter-registry.md); using one to create
a container is done with [otter create](commands/otter-create.md)'s
`--image` flag (or `images.default` in
[configuration.md](configuration.md)).

## Referring to an image

Anywhere otter accepts an image (`otter create --image`, manifest
`image` fields, `otter registry pull/remove`), you can use either:

- **A short registry name** — one of the names in the table below
  (case insensitive), resolved against the registry catalog.
- **A full image reference** — anything containing `/` or `:`, used
  exactly as given, bypassing the registry entirely (e.g. your own
  custom image, or a distro image not in the built-in catalog).

## Built-in images

This is the image actually pulled when you use a given short name
(the `official_image` for that entry) — i.e. what you get when the
entry is enabled. See [Enabled vs. disabled entries](#enabled-vs-disabled-entries)
below for what you get instead if it's disabled.

| Short name            | Image pulled by short name                    | Architectures |
| --------------------- | --------------------------------------------- | ------------- |
| `ubuntu`              | `ghcr.io/ferret-linux/ubuntu-otr:stable`      | amd64, arm64  |
| `ubuntu-lts`          | `ghcr.io/ferret-linux/ubuntu-otr:lts`         | amd64, arm64  |
| `debian`              | `ghcr.io/ferret-linux/debian-otr:stable`      | amd64, arm64  |
| `debian-testing`      | `ghcr.io/ferret-linux/debian-otr:testing`     | amd64, arm64  |
| `debian-unstable`     | `ghcr.io/ferret-linux/debian-otr:unstable`    | amd64, arm64  |
| `fedora`              | `ghcr.io/ferret-linux/fedora-otr:stable`      | amd64, arm64  |
| `fedora-rawhide`      | `ghcr.io/ferret-linux/fedora-otr:rawhide`     | amd64, arm64  |
| `arch`                | `ghcr.io/ferret-linux/arch-otr:latest`        | amd64         |
| `blackarch`           | `ghcr.io/ferret-linux/blackarch-otr:latest`   | amd64         |
| `gentoo`              | `ghcr.io/ferret-linux/gentoo-otr:stage3`      | amd64, arm64  |
| `alma`                | `ghcr.io/ferret-linux/alma-otr:stable`        | amd64, arm64  |
| `rocky`               | `ghcr.io/ferret-linux/rocky-otr:stable`       | amd64, arm64  |
| `centos`              | `ghcr.io/ferret-linux/centos-otr:stable`      | amd64, arm64  |
| `oracle`              | `ghcr.io/ferret-linux/oracle-otr:stable`      | amd64, arm64  |
| `rhel`                | `ghcr.io/ferret-linux/rhel-otr:stable`        | amd64, arm64  |
| `opensuse-leap`       | `ghcr.io/ferret-linux/opensuse-otr:leap`      | amd64, arm64  |
| `opensuse-tumbleweed` | `ghcr.io/ferret-linux/opensuse-otr:tumbleweed`| amd64, arm64  |
| `alpine`              | `ghcr.io/ferret-linux/alpine-otr:latest`      | amd64, arm64  |
| `alpine-edge`         | `ghcr.io/ferret-linux/alpine-otr:edge`        | amd64, arm64  |
| `kali`                | `ghcr.io/ferret-linux/kali-otr:rolling`       | amd64, arm64  |
| `kali-edge`           | `ghcr.io/ferret-linux/kali-otr:edge`          | amd64, arm64  |
| `void-glibc`          | `ghcr.io/ferret-linux/void-otr:glibc`         | amd64, arm64  |
| `void-musl`           | `ghcr.io/ferret-linux/void-otr:musl`          | amd64, arm64  |
| `chimera`             | `ghcr.io/ferret-linux/chimera-otr:latest`     | amd64, arm64  |
| `devuan`              | `ghcr.io/ferret-linux/devuan-otr:stable`      | amd64         |
| `devuan-testing`      | `ghcr.io/ferret-linux/devuan-otr:testing`     | amd64         |
| `devuan-unstable`     | `ghcr.io/ferret-linux/devuan-otr:unstable`    | amd64         |
| `wolfi`               | `ghcr.io/ferret-linux/wolfi-otr:latest`       | amd64, arm64  |

## Base images

Each entry above is built from an upstream base image via a
`Containerfile` under `images/` in the repository. RHEL family
distros (Alma, Rocky, CentOS, Oracle) share a common
`rhel-family.Containerfile` base; RHEL itself uses its own
`rhel.Containerfile`.

| Short name            | Containerfile               | Built from                                         |
| --------------------- | --------------------------- | -------------------------------------------------- |
| `ubuntu`              | `ubuntu.Containerfile`      | `docker.io/library/ubuntu:latest`                  |
| `ubuntu-lts`          | `ubuntu.Containerfile`      | `docker.io/library/ubuntu:26.04`                   |
| `debian`              | `debian.Containerfile`      | `docker.io/library/debian:stable`                  |
| `debian-testing`      | `debian.Containerfile`      | `docker.io/library/debian:testing`                 |
| `debian-unstable`     | `debian.Containerfile`      | `docker.io/library/debian:unstable`                |
| `fedora`              | `fedora.Containerfile`      | `quay.io/fedora/fedora:44`                         |
| `fedora-rawhide`      | `fedora.Containerfile`      | `quay.io/fedora/fedora:rawhide`                    |
| `arch`                | `archlinux.Containerfile`   | `docker.io/library/archlinux:latest`               |
| `blackarch`           | `blackarch.Containerfile`   | `docker.io/blackarchlinux/blackarch:latest`        |
| `gentoo`              | `gentoo.Containerfile`      | `docker.io/gentoo/stage3:latest`                   |
| `alma`                | `rhel-family.Containerfile` | `docker.io/library/almalinux:latest`               |
| `rocky`               | `rhel-family.Containerfile` | `docker.io/rockylinux/rockylinux:10-ubi-init`      |
| `centos`              | `rhel-family.Containerfile` | `quay.io/centos/centos:stream10`                   |
| `oracle`              | `rhel-family.Containerfile` | `container-registry.oracle.com/os/oraclelinux:10`  |
| `rhel`                | `rhel.Containerfile`        | `registry.access.redhat.com/ubi10/ubi-init:latest` |
| `opensuse-leap`       | `opensuse.Containerfile`    | `registry.opensuse.org/opensuse/leap:latest`       |
| `opensuse-tumbleweed` | `opensuse.Containerfile`    | `registry.opensuse.org/opensuse/tumbleweed:latest` |
| `alpine`              | `alpine.Containerfile`      | `docker.io/library/alpine:latest`                  |
| `alpine-edge`         | `alpine.Containerfile`      | `docker.io/library/alpine:edge`                    |
| `kali`                | `kali.Containerfile`        | `docker.io/kalilinux/kali-rolling:latest`          |
| `kali-edge`           | `kali.Containerfile`        | `docker.io/kalilinux/kali-bleeding-edge:latest`    |
| `void-glibc`          | `void.Containerfile`        | `ghcr.io/void-linux/void-glibc-full:latest`        |
| `void-musl`           | `void.Containerfile`        | `ghcr.io/void-linux/void-musl-full:latest`         |
| `chimera`             | `chimera.Containerfile`     | `docker.io/chimeralinux/chimera:latest`            |
| `devuan`              | `devuan.Containerfile`      | `docker.io/devuan/devuan:stable`                   |
| `devuan-testing`      | `devuan.Containerfile`      | `docker.io/devuan/devuan:testing`                  |
| `devuan-unstable`     | `devuan.Containerfile`      | `docker.io/devuan/devuan:unstable`                 |
| `wolfi`               | `wolfi.Containerfile`       | `cgr.dev/chainguard/wolfi-base:latest`             |

Note: the "Built from" image in this table is also the **fallback
vendor image** — the exact image you get if that entry is disabled
upstream (see below).

Run `otter registry list --all` for the live, up-to-date view of this
catalog, including any images that have since been disabled and any
that have been added upstream since these docs were written.

## Enabled vs. disabled entries

Each registry entry can be independently enabled or disabled
upstream. A disabled entry still resolves by short name, but to its
**fallback vendor image** — the distro's plain official image —
rather than otter's optimized build. `otter registry list` hides
disabled entries by default; pass `--all` to see them along with
their status.

## Image staleness

Every otter-built image bakes in an `otter.image_build` OCI label
carrying a monotonically increasing build number. When you already
have an image pulled locally, otter compares your local build number
against the latest one recorded in the registry manifest:

- **Current** — local build matches the latest known remote build;
  nothing happens.
- **Behind** — the remote build is ahead. If the gap reaches
  `images.staleness-warn-threshold` (from
  [otter.conf](configuration.md)), otter warns. If it reaches
  `images.staleness-autopull-threshold`, otter pulls the newer image
  automatically instead of just warning.
- **Ahead** — your local build is somehow newer than the latest known
  remote build (not expected in normal use); otter just warns.
- **Unknown** — the local image is missing the build label entirely
  (e.g. an older image, or one pulled before this feature existed);
  otter treats this like a missing image and pulls to heal it.

Both thresholds default to values in `pkg/config/defaults.go`
(`staleness-warn-threshold = 5`, `staleness-autopull-threshold = 10`)
but can be tuned or disabled (`0`) in `otter.conf`. This check runs
both when creating a container from an existing local image, and when
running `otter registry list` (see [otter-registry](commands/otter-registry.md)).

## See Also

- [otter-registry](commands/otter-registry.md) — browsing, pulling,
  and removing images.
- [otter-create](commands/otter-create.md) — using an image to create
  a container.
- [configuration.md](configuration.md) — setting a default image and
  staleness thresholds.
