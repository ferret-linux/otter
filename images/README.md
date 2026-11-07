# Otter images

This directory contains the Containerfiles and catalog data for Otter's
curated, Otter-ready container images. The images add a consistent base
configuration and packages to upstream distribution images. They are built
and published to GitHub Container Registry by the repository's image
workflows.

## Files

| File or directory | Purpose |
| --- | --- |
| `*.Containerfile` | Distribution-specific build recipes. Several catalog variants may use the same recipe. |
| `scripts/` | Shared setup, package validation, shell installation, and other image build helpers. |
| `images.json` | Build matrix: base image, Containerfile, published image name and tag, display name, architectures, and whether the target is enabled. |
| `images-properties.json` | Published-image metadata used by Otter's registry commands, including image references, fallback images, architectures, build number, and size data. |

These JSON files serve different purposes. `images.json` drives the image
build workflow. `images-properties.json` records the published images and is
fetched by the Otter CLI at runtime for its image registry catalog.

## Image build workflow

The [`Image Builder` workflow](../.github/workflows/image-builder.yml) runs on
a schedule and can also be started manually. It builds enabled entries from
`images.json` for their configured architectures, assembles and verifies the
multi-architecture image, and updates `images-properties.json` with the
published image metadata.

Published references use the form:

```text
ghcr.io/ferret-linux/<name>:<tag>
```

For example, the `ubuntu-otr` image's `stable` tag is
`ghcr.io/ferret-linux/ubuntu-otr:stable`.

## Adding or changing an image

1. Add or update the image entry in `images.json`. Keep its `containerfile`,
   upstream `image`, published `name` and `tag`, `arches`, and `enabled` value
   consistent with the recipe and desired build.
2. Create or update the referenced `*.Containerfile`. Reuse scripts from
   `scripts/` for setup shared across distributions, and keep distro-specific
   package names and setup in the appropriate recipe.
3. Check that every architecture listed in the catalog is supported by the
   upstream base image and the build recipe.
4. Run the image builder for a controlled change and verify the resulting
   image before relying on its generated registry metadata.

`images-properties.json` is updated by the successful image workflow. Avoid
hand-editing build numbers, sizes, or timestamps as part of a normal recipe
change; those values are collected from the published image.

For the runtime meaning of image short names, fallback images, and custom
image references, see Otter's [image documentation](../otter/pkg/commands/assets/docs/04.Concepts/01.Images.md).
