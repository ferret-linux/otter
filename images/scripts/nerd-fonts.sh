#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: ./nerd-fonts.sh
# Downloads and installs the latest NerdFontsSymbolsOnly into the container image

set -e

# Install NerdFontsSymbolsOnly (symbols-only nerd font, no base font needed)
# Fetches the latest release version from GitHub automatically
NERD_FONT_VERSION="$(curl -fsSL 'https://api.github.com/repos/ryanoasis/nerd-fonts/releases/latest' \
    | grep '"tag_name"' \
    | head -n1 \
    | cut -d'"' -f4)"

NERD_FONT_DIR="/usr/share/fonts/nerd-fonts"
NERD_FONT_URL="https://github.com/ryanoasis/nerd-fonts/releases/download/${NERD_FONT_VERSION}/NerdFontsSymbolsOnly.tar.xz"
NERD_FONT_TMP="$(mktemp -d)"

mkdir -p "${NERD_FONT_DIR}"
curl -fsSL -o "${NERD_FONT_TMP}/NerdFontsSymbolsOnly.tar.xz" "${NERD_FONT_URL}"
tar -xJ -C "${NERD_FONT_DIR}" -f "${NERD_FONT_TMP}/NerdFontsSymbolsOnly.tar.xz" --wildcards '*.ttf'

# Cleanup
rm -rf "${NERD_FONT_TMP}"