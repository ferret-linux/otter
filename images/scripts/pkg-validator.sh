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
packages=""
parsing_packages=0
next_is_pm=0

for arg in "$@"; do
    if [ "${arg}" = "--pkgmgr" ]; then
        next_is_pm=1
    elif [ "${next_is_pm}" = "1" ]; then
        pkgmgr="${arg}"
        next_is_pm=0
    elif [ "${arg}" = "--" ]; then
        parsing_packages=1
    elif [ "${parsing_packages}" = "1" ]; then
        packages="${packages} ${arg}"
    fi
done

if [ -z "${pkgmgr}" ]; then
    echo "ERROR: --pkgmgr is required" >&2
    exit 1
fi

if [ -z "${packages}" ]; then
    echo "ERROR: no packages provided after --" >&2
    exit 1
fi

valid=""
YELLOW='\033[1;33m'
RED='\033[0;31m'
RESET='\033[0m'

case "${pkgmgr}" in
    apk)
        # shellcheck disable=SC2086
        found="$(apk search -qe ${packages} 2>/dev/null | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    apt)
        # shellcheck disable=SC2086
        found="$(apt-cache show ${packages} 2>/dev/null | grep "^Package:" | sort -u | cut -d' ' -f2- | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    dnf)
        # shellcheck disable=SC2086
        found="$(dnf list -q ${packages} 2>/dev/null | \
            grep -v "Packages" | \
            grep "$(uname -m)\|noarch" | \
            cut -d' ' -f1 | \
            sed 's/\.[^.]*$//' | \
            tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    pacman)
        # shellcheck disable=SC2046,SC2086
        valid="$(pacman -Ssq 2>/dev/null | grep -E "^($(echo ${packages} | tr ' ' '|'))$" | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${valid} " in
                *" ${pkg} "*) ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    zypper)
        # shellcheck disable=SC2086
        found="$(zypper -n -q se --match-exact ${packages} 2>/dev/null | grep -e 'package$' | cut -d'|' -f2 | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    xbps)
        found="$(xbps-query -Rs '*' 2>/dev/null | awk '{print $2}' | sed 's/-[^-]*$//' | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) printf "${YELLOW}WARN: ${RED}%s${YELLOW} not found in repos${RESET}\n" "${pkg}" >&2 ;;
            esac
        done
        ;;
    emerge)
        for pkg in ${packages}; do
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
        valid="${packages}"
        ;;
    *)
        echo "ERROR: unknown package manager: ${pkgmgr}" >&2
        exit 1
        ;;
esac

echo "${valid}"
