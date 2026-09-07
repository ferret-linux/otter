# bevy

A terminal emulator for GNU/Linux, forked from
[Ptyxis](https://gitlab.gnome.org/chergert/ptyxis) — GNOME's container-oriented
terminal — and tailored for **otter integration** with **built-in Nerd Font
support** and other goodies on top of Ptyxis.

## What is this?

Ptyxis is an excellent, modern, GTK4/libadwaita based terminal designed around
containers (Podman, Toolbx, distrobox, SSH). `bevy` keeps all of that and adds:

- **Otter integration** — first-class integration with otter, supercharging
  the terminal workflow targeted by this fork.
- **Nerd Font support baked in** — proper rendering out of the box for the
  icon set used by Nerd Fonts (powerline glyphs, git/editor/programming icons,
  and friends), no manual font patching or tedious configuration required.
- **Other goodies** — quality-of-life tweaks and defaults on top of stock
  Ptyxis.

## Building

Building uses [Meson](https://mesonbuild.com/) (>= 1.0.0), exactly like upstream.
A convenience `Makefile` wraps the documented commands:

```sh
make check-deps     # verify all build dependencies are installed (distro-independent)
make build          # dev build (--buildtype=debug) and compile into _build
make build-release  # release build (--buildtype=release) and compile into _build-release
make install        # install the dev build to /usr/local
make install-release  # install the release build to /usr/local
make uninstall        # uninstall the dev build
make uninstall-release  # uninstall the release build
```

Optional variables: `make build PREFIX=/usr BUILDTYPE=plain MESON_OPTS="-Ddevelopment=true"`.

Alternatively, use Meson directly:

```sh
# dev build
meson setup _build --prefix=/usr/local --buildtype=debug
meson compile -C _build
meson install -C _build

# release build
meson setup _build-release --prefix=/usr/local --buildtype=release
meson compile -C _build-release
meson install -C _build-release
```

See [Ptyxis' README](https://gitlab.gnome.org/chergert/ptyxis/-/blob/main/README.md)
for the full upstream build instructions.

## Credits

Ptyxis is created and maintained by the
[GNOME community](https://gitlab.gnome.org/chergert/ptyxis), primarily by
Christian Hergert and contributors, and is licensed under the GPL-3.0.
`bevy` is a fork of Ptyxis — huge thanks to all the upstream authors and
maintainers for their work.

This fork is maintained by itzNOELdev. Upstream copyright notices are kept
intact throughout the source tree; see [NOTICE.md](NOTICE.md) for details.
