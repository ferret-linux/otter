# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors

ARG IMAGE
FROM ${IMAGE}

ARG OTTER_BUILD_NUMBER
LABEL otter.image_build=${OTTER_BUILD_NUMBER}

# Pre-create otter dirs
COPY images/scripts/setup-common.sh /tmp/setup-common.sh
RUN sh /tmp/setup-common.sh

# Add otter image identifiers
RUN touch /usr/lib/otter/container.slackware

# NOTE: no mirror-selection step needed here - aclemons/slackware
# ships with slackpkg already pointed at the correct mirror for
# whichever release this tag actually is ("slackpkg needs to work
# out of the box" is one of that image's stated goals).

# Run non-interactively (should we make it default?)
#RUN sed -i 's/^DIALOG=.*/DIALOG=off/' /etc/slackpkg/slackpkg.conf 2>/dev/null || \
#    echo "DIALOG=off" >> /etc/slackpkg/slackpkg.conf

COPY images/scripts/pkg-validator.sh /tmp/pkg-validator.sh

# Bootstrap the network/package tools required before installing Slackpkg+.
# Slackpkg does not resolve dependencies, so include curl's runtime
# dependencies explicitly. git/wget are also bootstrapped here because
# they are needed by the image's subsequent package/tooling setup.
RUN printf 'yes\n' | DIALOG=off slackpkg update gpg && \
    DIALOG=off slackpkg update && \
    for pkg in $(sh /tmp/pkg-validator.sh --pkgmgr slackpkg -- \
        curl \
        git \
        wget \
        zstd \
        libpsl \
        libssh2 \
        libidn2 \
        nghttp2); do \
        DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install "${pkg}"; \
    done

# Install the latest Slackpkg+ release from alienbob.
# GitHub releases contain a pre-built Slackware .txz package.
# installpkg is the upstream-recommended installation method.
RUN set -eux; \
    tmp="$(mktemp -d)"; \
    pkg_url="$(curl -fsSL \
      https://api.github.com/repos/alienbob/slackpkgplus/releases/latest \
      | grep -o '"browser_download_url": "[^"]*\.txz"' \
      | head -1 \
      | cut -d'"' -f4)"; \
    [ -n "$pkg_url" ]; \
    pkg="${pkg_url##*/}"; \
    echo "Installing Slackpkg+: ${pkg}"; \
    curl -fL -o "$tmp/$pkg" "$pkg_url"; \
    installpkg "$tmp/$pkg"; \
    rm -rf "$tmp"

# NOTE: slackpkg does not perform automatic dependency resolution -
# unlike apt/dnf/pacman, packages not explicitly listed here will
# not be pulled in as transitive dependencies. This list may need
# to be more exhaustive than other distros' equivalents.
#
# NOTE: default BSD-style init (/etc/rc.d, sysvinit package) is
# already present as part of Slackware's mandatory base "A" series -
# nothing installed or changed here. systemd is intentionally not
# in this list (not part of Slackware)
#
# NOTE: this list is NOT guaranteed exhaustive for the multimedia/
# graphics/portal stack (ffmpeg, gstreamer plugins, mesa, pipewire,
# wireplumber, xdg-desktop-portal, xwayland). Those packages' exact
# runtime deps vary by how each was actually built (enabled codecs,
# GPU backends, etc.) and can't be reliably guessed from docs alone.
# The ldd verification step below is the source of truth for those -
# run the build, and if it fails, feed the reported missing .so
# names back in here as newly-identified packages.
RUN printf 'yes\n' | DIALOG=off slackpkg update gpg && \
    DIALOG=off slackpkg update && \
    for pkg in $(sh /tmp/pkg-validator.sh --pkgmgr slackpkg -- \
        bc \
        xz \
        mtr \
        tar \
        zsh \
        bash \
        fish \
        gzip \
        lame \
        less \
        lsof \
        mesa \
        sudo \
        time \
        tree \
        bzip2 \
        gnupg \
        rsync \
        unzip \
        which \
        dialog \
        ffmpeg \
        libdrm \
        libvpx \
        man-db \
        shadow \
        tzdata \
        infozip \
        ncurses \
        openssh \
        python3 \
        tcpdump \
        wayland \
        hostname \
        iproute2 \
        keyutils \
        libglvnd \
        libinput \
        nss-mdns \
        pinentry \
        pipewire \
        diffutils \
        findutils \
        gstreamer \
        procps-ng \
        util-linux \
        wireplumber \
        bash-completion \
        gst-plugins-base \
        gst-plugins-good \
        gst-plugins-libav \
        wayland-protocols \
        xdg-desktop-portal \
        gst-plugins-bad-free \
        xorg-server-xwayland); do \
        DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install "${pkg}"; \
    done

# Verify every installed ELF binary/library actually resolves its
# shared-library deps - catches slackpkg's lack of dep resolution
# instead of discovering it one broken binary at a time in CI.
#
# This is the authoritative check for anything not covered above
# (ffmpeg codecs, gstreamer plugins, mesa/GL, pipewire, etc.) - if
# it fails, the missing .so names it prints map 1:1 to the package
# that needs to be added to the list above.
RUN missing=0; \
    for f in $(find /usr/bin /usr/sbin /usr/lib* -xtype f 2>/dev/null); do \
      if file "$f" 2>/dev/null | grep -q ELF; then \
        out="$(ldd "$f" 2>&1)"; \
        if echo "$out" | grep -q "not found"; then \
          echo "=== $f ==="; \
          echo "$out" | grep "not found"; \
          missing=1; \
        fi; \
      fi; \
    done; \
    [ "$missing" -eq 0 ] || (echo "ERROR: missing shared library dependencies detected above" && exit 1)

# Fish is not in official slackware repos at the moment
RUN if [ -x /usr/bin/fish ]; then \
      echo "fish: already installed ($(/usr/bin/fish --version))"; \
    else \
      echo "fish: not installed, fetching latest AMD64 release"; \
      set -eux; \
      tmp="$(mktemp -d)"; \
      url="$(curl -fsSL https://api.github.com/repos/fish-shell/fish-shell/releases/latest \
        | grep -o '"browser_download_url": "[^"]*linux-x86_64\.tar\.xz"' \
        | head -1 \
        | cut -d'"' -f4)"; \
      [ -n "$url" ]; \
      curl -fL -o "$tmp/fish.tar.xz" "$url"; \
      tar -xJf "$tmp/fish.tar.xz" -C "$tmp"; \
      install -m 0755 "$tmp/fish" /usr/bin/fish; \
      echo "fish: installed $(/usr/bin/fish --version)"; \
      rm -rf "$tmp"; \
    fi

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh