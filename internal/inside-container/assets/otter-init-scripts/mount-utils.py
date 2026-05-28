#!/usr/bin/env python3
# shellcheck disable=all
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the otter project:
#    https://github.com/ferret-project/otter
# Copyright (C) 2026 otter contributors
#
# This file is based on distrobox:
#    https://github.com/89luca89/distrobox
# Copyright (C) 2021 distrobox contributors
#
# otter is free software; you can redistribute it and/or modify it
# under the terms of the GNU General Public License version 3
# as published by the Free Software Foundation.
#
# otter is distributed in the hope that it will be useful, but
# WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
# General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with otter; if not, see <http://www.gnu.org/licenses/>.

import os
import re
import shutil
import subprocess
from pathlib import Path


def get_host_locale():
    host_locale = ""
    host_locale_encoding = ""
    host_locale_lang = ""

    for locale_file in ["/run/host/etc/locale.conf", "/run/host/etc/default/locale"]:
        if Path(locale_file).is_file():
            for line in Path(locale_file).read_text().splitlines():
                if line.startswith("LANG="):
                    host_locale = line.removeprefix("LANG=").strip().strip('"').strip("'")
                    break
            if host_locale:
                m = re.match(r'^([^.]+)\.(.+)$', host_locale)
                if m:
                    host_locale_lang = m.group(1)
                    host_locale_encoding = m.group(2)
                break

    if not host_locale or host_locale == "C.UTF-8":
        host_locale = "en_US.UTF-8"
        host_locale_encoding = "UTF-8"
        host_locale_lang = "en_US"

    return host_locale, host_locale_encoding, host_locale_lang


def get_locked_mount_flags(src):
    if not shutil.which("findmnt"):
        return ""

    src = Path(src)
    if not src.exists():
        return ""

    locked_flags = []
    prev = None

    while True:
        result = subprocess.run(
            ["findmnt", "--noheadings", "--output", "OPTIONS", "--target", str(src)],
            capture_output=True, text=True
        )
        flags = result.stdout.strip()
        if flags:
            break
        prev = src
        src = src.parent
        if src == prev:
            return ""

    for flag in ["nodev", "noexec", "nosuid"]:
        if flag in flags:
            locked_flags.append(flag)

    return ",".join(locked_flags)


def init_readlink(path):
    result = subprocess.run(["ls", "-l", str(path)], capture_output=True, text=True)
    m = re.search(r'-> (.+)', result.stdout)
    if not m:
        return str(path)
    target = m.group(1).strip()
    target = target.replace("../", "/")
    return target


def mount_bind(source_dir, target_dir, mount_flags=""):
    source_dir = Path(source_dir)
    target_dir = Path(target_dir)

    if source_dir.is_symlink():
        source_dir = Path(init_readlink(source_dir))
        if "/run/host" not in str(source_dir):
            source_dir = Path("/run/host") / str(source_dir).lstrip("/")

    if not source_dir.is_dir() and not source_dir.is_file():
        return True

    if target_dir.exists() and subprocess.run(
        ["findmnt", str(target_dir)], capture_output=True
    ).returncode == 0:
        subprocess.run(["umount", str(target_dir)])

    if target_dir.is_symlink():
        target_dir.unlink()

    if source_dir.is_dir():
        if not target_dir.exists():
            try:
                target_dir.mkdir(parents=True, exist_ok=True)
            except Exception:
                print(f"Warning: cannot create mount target directory: {target_dir}")
                return False
    elif source_dir.is_file():
        if not target_dir.parent.is_dir():
            target_dir.parent.mkdir(parents=True, exist_ok=True)
        try:
            target_dir.touch()
        except Exception:
            print(f"Warning: cannot create mount target file: {target_dir}")
            return False

    if not mount_flags:
        if subprocess.run(["mount", "--rbind", str(source_dir), str(target_dir)]).returncode != 0:
            print(f"Warning: failed to bind mount {source_dir} to {target_dir}")
            return False
        if subprocess.run(["mount", "--make-rslave", str(target_dir)]).returncode != 0:
            print(f"Warning: failed to make rslave to {target_dir}")
            return False
    else:
        if subprocess.run(["mount", "--rbind", "-o", mount_flags, str(source_dir), str(target_dir)]).returncode != 0:
            print(f"Warning: failed to bind mount {source_dir} to {target_dir} using option {mount_flags}")
            return False

    return True