#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-linux/otter
# Copyright (C) 2026 otter contributors
#
# Usage: pkg-validator.sh --pkgmgr <pm> -- pkg1 pkg2 ...
# Validates packages exist in repos, prints valid list to stdout.
# Warns to stderr for any missing packages.

set -e

pkgmgr=""

# Parse options before --, then leave the package list in the positional
# parameters so it stays an array of distinct arguments.
while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do
    case "$1" in
        --pkgmgr)
            shift
            if [ "$#" -eq 0 ]; then
                echo "ERROR: --pkgmgr requires a value" >&2
                exit 1
            fi
            pkgmgr="$1"
            ;;
    esac
    shift
done

if [ "$#" -gt 0 ] && [ "$1" = "--" ]; then
    shift
fi

if [ -z "${pkgmgr}" ]; then
    echo "ERROR: --pkgmgr is required" >&2
    exit 1
fi

if [ "$#" -eq 0 ]; then
    echo "ERROR: no packages provided after --" >&2
    exit 1
fi

valid=""
YELLOW='\033[1;33m'
RED='\033[0;31m'
RESET='\033[0m'

case "${pkgmgr}" in
    apk)
        found="$(apk search -qe "$@" 2>/dev/null | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    apt)
        found="$(apt-cache show "$@" 2>/dev/null | grep "^Package:" | sort -u | cut -d' ' -f2- | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    brew)
        pattern=""
        for pkg in "$@"; do
            pattern="${pattern}|${pkg}"
        done
        pattern="${pattern#|}"
        found="$(brew formulae 2>/dev/null | grep -E "^(${pattern})$" | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    dnf)
        found="$(dnf list -q "$@" 2>/dev/null | \
            grep -v "Packages" | \
            grep "$(uname -m)\|noarch" | \
            cut -d' ' -f1 | \
            sed 's/\.[^.]*$//' | \
            tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    pacman)
        pattern=""
        for pkg in "$@"; do
            pattern="${pattern}|${pkg}"
        done
        pattern="${pattern#|}"
        valid="$(pacman -Ssq 2>/dev/null | grep -E "^(${pattern})$" | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${valid} " in
                *" ${pkg} "*) ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    zypper)
        found="$(zypper -n -q se --match-exact "$@" 2>/dev/null | grep -e 'package$' | cut -d'|' -f2 | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    xbps)
        found="$(xbps-query -Rs '*' 2>/dev/null | awk '{print $2}' | sed 's/-[^-]*$//' | tr '\n' ' ')"
        for pkg in "$@"; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    emerge)
        for pkg in "$@"; do
            if emerge --ask=n --search "${pkg}" 2>/dev/null | grep "Applications found" | grep -qE "[1-9][0-9]*"; then
                valid="${valid} ${pkg}"
            else
                printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2
            fi
        done
        ;;
    slackpkg)
        # No pre-install search/validation here by design:
        #  1. slackpkg search calls done here would run inside the
        #     same command substitution as the real install, ahead of
        #     any lock clearing done by the caller - a search that
        #     exits abnormally (e.g. SIGPIPE'd by a `grep -q` match)
        #     can leave slackpkg's own lock file behind and block the
        #     install that immediately follows.
        #  2. slackpkg's search output format is not reliable to
        #     parse: it differs between plain slackpkg and slackpkg+
        #     (multi-repo) installs.
        #  3. That unreliability is worse once third-party repos are
        #     layered in via slackpkg+ - official and third-party
        #     listings don't share one consistent format to match
        #     against.
        #  4. slackpkg install already no-ops safely on a pattern
        #     that matches nothing ("No packages match the pattern
        #     for install."), so this step was never required for
        #     safety in the first place.
        # The package list is passed straight through unchanged.
        for pkg in "$@"; do
            valid="${valid} ${pkg}"
        done
        ;;
    *)
        echo "ERROR: unknown package manager: ${pkgmgr}" >&2
        exit 1
        ;;
esac

echo "${valid}"
