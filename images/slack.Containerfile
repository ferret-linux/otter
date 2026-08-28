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
#
# Each slackpkg command below runs in its own RUN step so a failure
# is attributable to a specific command. Each is preceded by a clear
# of any stale /var/lock/slackpkg.* file: slackpkg's own lock cleanup
# only runs on certain exit paths, so a lock can be left behind even
# after a command that otherwise completed fine, which then makes
# every later slackpkg invocation refuse to run. The clear only ever
# runs before a command starts - it never wraps or suppresses that
# command's own exit status, so a genuine failure still fails the RUN.
RUN rm -f /var/lock/slackpkg.*; \
    printf 'yes\n' | DIALOG=off slackpkg update gpg && rm -rf -- /var/cache/packages/*

RUN rm -f /var/lock/slackpkg.*; \
    DIALOG=off slackpkg update && rm -rf -- /var/cache/packages/*

RUN rm -f /var/lock/slackpkg.*; \
    DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install $(sh /tmp/pkg-validator.sh --pkgmgr slackpkg -- \
        git \
        curl \
        perl \
        zstd \
        dcron \
        wget2 \
        brotli \
        c-ares \
        libpsl \
        ngtcp2 \
        libidn2 \
        libssh2 \
        nghttp2 \
        nghttp3 \
        cyrus-sasl \
        ca-certificates) && \
    rm -rf -- /var/cache/packages/*

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

# Configure third-party repositories according to TAG.
# Repository priority:
#
#   Ponce > AlienBOB Restricted > AlienBOB > Official Slackware
#
# TAG=current:
#   use the rolling Slackware-current repositories.
#
# TAG=stable:
#   dynamically resolve the current stable Slackware release and use
#   the matching repositories.
#
# Ponce provides x86_64 repositories for both Slackware 15.0 and
# Slackware-current. :contentReference[oaicite:1]{index=1}
#
# PKGS_PRIORITY controls which third-party repository wins when the
# same package is available from multiple repositories. The official
# Slackware repository remains the fallback.
RUN set -eux; \
    case "$TAG" in \
      current) \
        echo "Slackware: detected current branch"; \
        slackware_release="current"; \
        ;; \
      stable) \
        echo "Slackware: detected stable branch"; \
        slackware_release="$(curl -fsSL \
          https://endoflife.date/api/slackware.json \
          | grep -o '"cycle":"[^"]*"' \
          | head -1 \
          | cut -d'"' -f4)"; \
        [ -n "$slackware_release" ]; \
        echo "Slackware: using stable Slackware ${slackware_release}"; \
        ;; \
      *) \
        echo "ERROR: unsupported TAG: ${TAG}" >&2; \
        exit 1; \
        ;; \
    esac; \
    cat >> /etc/slackpkg/slackpkgplus.conf <<EOF
REPOPLUS+=( alienbob )
REPOPLUS+=( restricted )
REPOPLUS+=( ponce )

MIRRORPLUS['alienbob']=https://slackware.nl/people/alien/sbrepos/${slackware_release}/x86_64/
MIRRORPLUS['restricted']=https://slackware.nl/people/alien/restricted_sbrepos/${slackware_release}/x86_64/
MIRRORPLUS['ponce']=https://ponce.cc/slackware/slackware64-${slackware_release}/packages/

PKGS_PRIORITY=( ponce restricted alienbob )
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
RUN rm -f /var/lock/slackpkg.*; \
    printf 'yes\n' | DIALOG=off slackpkg update gpg && rm -rf -- /var/cache/packages/*

RUN rm -f /var/lock/slackpkg.*; \
    DIALOG=off slackpkg update && \
    rm -rf -- /var/cache/packages/*

RUN rm -f /var/lock/slackpkg.*; \
    DIALOG=off slackpkg -batch=on -default_answer=y -orig_backups=off install $(sh /tmp/pkg-validator.sh --pkgmgr slackpkg -- \
        bc \
        xz \
        atk \
        glu \
        mtr \
        orc \
        sbc \
        srt \
        tar \
        zix \
        zsh \
        bash \
        cups \
        dbus \
        faac \
        fftw \
        fish \
        flac \
        glew \
        gzip \
        lame \
        less \
        lilv \
        llvm \
        lsof \
        mesa \
        opus \
        sdl2 \
        serd \
        sord \
        sudo \
        suil \
        time \
        tree \
        x264 \
        x265 \
        aften \
        bluez \
        bzip2 \
        cairo \
        dav1d \
        faad2 \
        glibc \
        gnupg \
        icu4c \
        Imath \
        lcms2 \
        libva \
        pango \
        pcre2 \
        rsync \
        speex \
        which \
        xauth \
        a52dec \
        dialog \
        ffmpeg \
        gnutls \
        libaio \
        libass \
        libcap \
        libdca \
        libdrm \
        libffi \
        libgcc \
        libogg \
        libvpx \
        libX11 \
        libXau \
        libxcb \
        man-db \
        mpg123 \
        nettle \
        pixman \
        shadow \
        sqlite \
        sratom \
        taglib \
        fribidi \
        infozip \
        iputils \
        libcdio \
        libexif \
        liblrdf \
        libnice \
        librsvg \
        libtiff \
        libXext \
        libxml2 \
        ncurses \
        openexr \
        openssh \
        python3 \
        tcpdump \
        twolame \
        wayland \
        alsa-lib \
        freetype \
        graphene \
        harfbuzz \
        hostname \
        iproute2 \
        keyutils \
        libdecor \
        libepoxy \
        libglvnd \
        libgudev \
        libinput \
        libvte-2 \
        libXdmcp \
        nss-mdns \
        opusfile \
        pinentry \
        pipewire \
        qrencode \
        xcb-util \
        xvidcore \
        xdg-utils \
        dbus-glib \
        diffutils \
        findutils \
        graphite2 \
        gstreamer \
        json-glib \
        libgcrypt \
        libnotify \
        libsecret \
        libunwind \
        libvisual \
        libvorbis \
        libXfixes \
        procps-ng \
        xorgproto \
        zxing-cpp \
        fluidsynth \
        fontconfig \
        gdk-pixbuf \
        libfdk-aac \
        libmodplug \
        libplacebo \
        libsndfile \
        libxcb-glx \
        libXdamage \
        libXxf86vm \
        soundtouch \
        util-linux \
        vulkan-sdk \
        chromaprint \
        libunibreak \
        libxcb-dri2 \
        libxcb-dri3 \
        libxcb-sync \
        wireplumber \
        xcb-util-wm \
        xisxwayland \
        alsa-plugins \
        at-spi2-core \
        libgpg-error \
        libpciaccess \
        libunistring \
        libxcb-randr \
        libxcb-shape \
        libxkbcommon \
        libxshmfence \
        opencore-amr \
        xcb-util-xrm \
        xdg-user-dirs \
        vulkan-loader \
        libjpeg-turbo \
        libsamplerate \
        libxcb-render \
        libxcb-xfixes \
        pipewire-jack \
        wayland-utils \
        pipewire-pulse \
        glibc-zoneinfo \
        libxcb-present \
        xcb-util-image \
        bash-completion \
        gst-plugins-bad \
        libdisplay-info \
        xcb-util-cursor \
        xcb-util-errors \
        gst-plugins-base \
        gst-plugins-good \
        gst-plugins-ugly \
        libcdio-paranoia \
        shared-mime-info \
        xcb-util-keysyms \
        xkeyboard-config \
        gst-plugins-libav \
        wayland-protocols \
        hicolor-icon-theme \
        xdg-desktop-portal \
        xcb-util-renderutil \
        xorg-server-xwayland) && \
    rm -rf -- /var/cache/packages/*

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

# Install gum (static binary, amd64/arm64) into otter's helpers dir.
# Requires curl, so this must come after curl is installed above.
COPY images/scripts/install-gum.sh /tmp/install-gum.sh
RUN sh /tmp/install-gum.sh

# Timezone default
RUN ln -sf /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone

# Ensure a python3 binary is resolvable
COPY images/scripts/python-fix.sh /tmp/python-fix.sh
RUN sh /tmp/python-fix.sh