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

case "${pkgmgr}" in
    apk)
        # shellcheck disable=SC2086
        found="$(apk search -qe ${packages} 2>/dev/null | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) echo "WARN: ${pkg} not found in repos" >&2 ;;
            esac
        done
        ;;
    apt)
        # shellcheck disable=SC2086
        found="$(apt-cache show ${packages} 2>/dev/null | grep "^Package:" | sort -u | cut -d' ' -f2- | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) echo "WARN: ${pkg} not found in repos" >&2 ;;
            esac
        done
        ;;
    dnf)
        # shellcheck disable=SC2086
        found="$(dnf list -q ${packages} 2>/dev/null | grep "$(uname -m)\|noarch" | cut -d' ' -f1 | sed 's/\.[^.]*$//')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) echo "WARN: ${pkg} not found in repos" >&2 ;;
            esac
        done
        ;;
    pacman)
        found="$(pacman -Ssq 2>/dev/null | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) echo "WARN: ${pkg} not found in repos" >&2 ;;
            esac
        done
        ;;
    zypper)
        for pkg in ${packages}; do
            if zypper -n -q se --match-exact "${pkg}" 2>/dev/null | grep -q 'package'; then
                valid="${valid} ${pkg}"
            else
                echo "WARN: ${pkg} not found in repos" >&2
            fi
        done
        ;;
    xbps)
        found="$(xbps-query -Rs '*' 2>/dev/null | awk '{print $2}' | sed 's/-[^-]*$//' | tr '\n' ' ')"
        for pkg in ${packages}; do
            case " ${found} " in
                *" ${pkg} "*) valid="${valid} ${pkg}" ;;
                *) echo "WARN: ${pkg} not found in repos" >&2 ;;
            esac
        done
        ;;
    emerge)
        for pkg in ${packages}; do
            if emerge --ask=n --search "${pkg}" 2>/dev/null | grep "Applications found" | grep -qE "[1-9][0-9]*"; then
                valid="${valid} ${pkg}"
            else
                echo "WARN: ${pkg} not found in repos" >&2
            fi
        done
        ;;
    *)
        echo "ERROR: unknown package manager: ${pkgmgr}" >&2
        exit 1
        ;;
esac

echo "${valid}"