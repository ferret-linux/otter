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

| Short name            | Upstream image thats used to make the otter image  | Architectures |
| --------------------- | -------------------------------------------------- | ------------- |
| `ubuntu`              | `docker.io/library/ubuntu:latest`                  | amd64, arm64  |
| `ubuntu-lts`          | `docker.io/library/ubuntu:26.04`                   | amd64, arm64  |
| `debian`              | `docker.io/library/debian:stable`                  | amd64, arm64  |
| `debian-testing`      | `docker.io/library/debian:testing`                 | amd64, arm64  |
| `debian-unstable`     | `docker.io/library/debian:unstable`                | amd64, arm64  |
| `fedora`              | `quay.io/fedora/fedora:44`                         | amd64, arm64  |
| `fedora-rawhide`      | `quay.io/fedora/fedora:rawhide`                    | amd64, arm64  |
| `arch`                | `docker.io/library/archlinux:latest`               | amd64         |
| `blackarch`           | `docker.io/blackarchlinux/blackarch:latest`        | amd64         |
| `gentoo`              | `docker.io/gentoo/stage3:latest`                   | amd64, arm64  |
| `alma`                | `docker.io/library/almalinux:latest`               | amd64, arm64  |
| `rocky`               | `docker.io/rockylinux/rockylinux:10-ubi-init`      | amd64, arm64  |
| `centos`              | `quay.io/centos/centos:stream10`                   | amd64, arm64  |
| `oracle`              | `container-registry.oracle.com/os/oraclelinux:10`  | amd64, arm64  |
| `rhel`                | `registry.access.redhat.com/ubi10/ubi-init:latest` | amd64, arm64  |
| `opensuse-leap`       | `registry.opensuse.org/opensuse/leap:latest`       | amd64, arm64  |
| `opensuse-tumbleweed` | `registry.opensuse.org/opensuse/tumbleweed:latest` | amd64, arm64  |
| `alpine`              | `docker.io/library/alpine:latest`                  | amd64, arm64  |
| `alpine-edge`         | `docker.io/library/alpine:edge`                    | amd64, arm64  |
| `kali`                | `docker.io/kalilinux/kali-rolling:latest`          | amd64, arm64  |
| `kali-edge`           | `docker.io/kalilinux/kali-bleeding-edge:latest`    | amd64, arm64  |
| `void-glibc`          | `ghcr.io/void-linux/void-glibc-full:latest`        | amd64, arm64  |
| `void-musl`           | `ghcr.io/void-linux/void-musl-full:latest`         | amd64, arm64  |
| `chimera`             | `docker.io/chimeralinux/chimera:latest`            | amd64, arm64  |
| `devuan`              | `docker.io/devuan/devuan:stable`                   | amd64         |
| `devuan-testing`      | `docker.io/devuan/devuan:testing`                  | amd64         |
| `devuan-unstable`     | `docker.io/devuan/devuan:unstable`                 | amd64         |
| `wolfi`               | `cgr.dev/chainguard/wolfi-base:latest`             | amd64, arm64  |

Each entry corresponds to a `Containerfile` under `images/` in the
repository (e.g. `fedora.Containerfile`, `alpine.Containerfile`) that
builds otter's optimized version of that distro's official image.
RHEL family distros (Alma, Rocky, CentOS, Oracle) share a common
`rhel-family.Containerfile` base.

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
