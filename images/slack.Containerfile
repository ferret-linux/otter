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
ARG TAG
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
        wget2 \
        zstd \
        libpsl \
        libssh2 \
        libidn2 \
        nghttp2 \
        nghttp3 \
        perl \
        ngtcp2 \
        ca-certificates \
        brotli \
        dcron \
        cyrus-sasl \
        c-ares); do \
        DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install "${pkg}"; \
    done

# Slackware's package tooling does not guarantee the CA trust store is
# refreshed after installing/upgrading ca-certificates. Rebuild the
# system CA bundle and OpenSSL hash links, then verify the resulting
# trust store with a real HTTPS request. This prevents TLS failures when
# subsequent build steps fetch packages or releases from HTTPS sources.
RUN set -eux; \
    /usr/sbin/update-ca-certificates; \
    /usr/sbin/update-ca-certificates -f; \
    openssl rehash /etc/ssl/certs; \
    test -s /etc/ssl/certs/ca-certificates.crt; \
    curl -fsSL https://api.github.com/ >/dev/null

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

# Configure AlienBOB repositories according to TAG.
# TAG=current:
#   use AlienBOB's rolling Slackware-current repositories.
#
# TAG=stable:
#   dynamically resolve the current stable Slackware release and use
#   the matching AlienBOB repositories. This avoids hardcoding a
#   release number here so the image automatically follows future
#   stable Slackware releases.
#
# Enabled repositories:
#   alienbob   - AlienBOB's main third-party binary packages.
#   restricted - AlienBOB's restricted/multimedia packages.
#
# PKGS_PRIORITY makes restricted and alienbob take precedence over
# the official Slackware repositories when the same package exists
# in multiple repositories.
RUN set -eux; \
    case "$TAG" in \
      current) \
        echo "AlienBOB: detected current branch"; \
        slackware_release="current"; \
        ;; \
      stable) \
        echo "AlienBOB: detected stable branch"; \
        slackware_release="$(curl -fsSL \
          https://endoflife.date/api/slackware.json \
          | grep -o '"cycle":"[^"]*"' \
          | head -1 \
          | cut -d'"' -f4)"; \
        [ -n "$slackware_release" ]; \
        echo "AlienBOB: using stable Slackware ${slackware_release}"; \
        ;; \
    esac; \
    cat >> /etc/slackpkg/slackpkgplus.conf <<EOF
REPOPLUS+=( alienbob )
REPOPLUS+=( restricted )

MIRRORPLUS['alienbob']=https://slackware.nl/people/alien/sbrepos/${slackware_release}/x86_64/
MIRRORPLUS['restricted']=https://slackware.nl/people/alien/restricted_sbrepos/${slackware_release}/x86_64/

PKGS_PRIORITY=( restricted alienbob )
EOF

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
        aften \
        faac \
        faad2 \
        ffmpeg \
        libdca \
        libdrm \
        libfdk-aac \
        libvpx \
        opencore-amr \
        twolame \
        x264 \
        x265 \
        xvidcore \
        man-db \
        shadow \
        tzdata \
        infozip \
        libaio \
        libcap \
        libffi \
        libgcc \
        libgcrypt \
        libgpg-error \
        libjpeg-turbo \
        libogg \
        libsndfile \
        atk \
        at-spi2-atk \
        at-spi2-core \
        cairo \
        cups \
        dbus-glib \
        fontconfig \
        freetype \
        fribidi \
        gdk-pixbuf \
        harfbuzz \
        hicolor-icon-theme \
        json-glib \
        libepoxy \
        libexif \
        libgudev \
        libnotify \
        librsvg \
        libsecret \
        libunwind \
        libxkbcommon \
        libxkbcommon-x11 \
        pango \
        shared-mime-info \
        wayland-utils \
        xcb-util-cursor \
        xcb-util-errors \
        xcb-util-xrm \
        libtiff \
        libunistring \
        libvorbis \
        libxml2 \
        opus \
        pcre2 \
        sqlite \
        speex \
        taglib \
        flac \
        mpg123 \
        orc \
        sdl2 \
        xcb-util \
        xcb-util-image \
        xcb-util-keysyms \
        xcb-util-renderutil \
        xcb-util-wm \
        xkeyboard-config \
        xorgproto \
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
        gst-plugins-ugly \
        gst-plugins-bad \
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
    [ "$missing" -eq 0 ] || echo "WARNING: missing shared library dependencies detected above"

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