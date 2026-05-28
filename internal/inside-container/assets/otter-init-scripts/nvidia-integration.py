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

import fnmatch
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from mount_utils import get_locked_mount_flags, mount_bind # type: ignore


def _find_files(base, patterns, exclude_patterns=None):
    base = Path(base)
    if not base.exists():
        return []
    results = []
    for path in base.rglob("*"):
        if path.is_dir():
            continue
        name = path.name
        full = str(path)
        if exclude_patterns and any(fnmatch.fnmatch(full, f"*{e}*") for e in exclude_patterns):
            continue
        if any(fnmatch.fnmatch(name, p) or fnmatch.fnmatch(full, f"*{p}*") for p in patterns):
            results.append(path)
    return results


def _mount_nvidia_file(nvidia_file, dest_file):
    dest_file = Path(dest_file)

    if not dest_file.parent.exists():
        try:
            dest_file.parent.mkdir(parents=True, exist_ok=True)
        except Exception:
            print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
            return
    if not dest_file.parent.stat().st_mode & 0o200:
        print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
        return

    nvidia_file = Path(nvidia_file)
    if nvidia_file.is_symlink():
        nvidia_file = Path(str(nvidia_file.resolve()))

    locked_flags = get_locked_mount_flags(str(nvidia_file))
    flags = "ro" + (f",{locked_flags}" if locked_flags else "")
    mount_bind(str(nvidia_file), str(dest_file), flags)


def setup_nvidia():
    print("otter: Setting up host's nvidia integration...")

    # Cleanup empty stale so/nvidia files
    for pattern, search_dirs in [
        (["*.so*"], ["/usr/lib*"]),
        (["*nvidia*"], ["/usr/", "/etc/"]),
    ]:
        for search_dir in search_dirs:
            for p in Path("/").glob(search_dir.lstrip("/")):
                if not p.exists():
                    continue
                for f in p.rglob("*"):
                    if f.is_dir():
                        continue
                    if any(fnmatch.fnmatch(f.name, pat) for pat in pattern) and f.stat().st_size == 0:
                        try:
                            f.unlink()
                        except Exception:
                            subprocess.run(["umount", str(f)], capture_output=True)
                            try:
                                f.unlink()
                            except Exception:
                                pass

    # Config files from /run/host/etc/
    for nvidia_file in _find_files("/run/host/etc", ["*nvidia*"], exclude_patterns=["/systemd/"]):
        dest_file = str(nvidia_file).replace("/run/host", "", 1)
        _mount_nvidia_file(nvidia_file, dest_file)

    # EGL/ICD/Vulkan confs from /run/host/usr/
    conf_patterns = [
        "*glvnd/egl_vendor.d/10_nvidia.json",
        "*X11/xorg.conf.d/10-nvidia.conf",
        "*X11/xorg.conf.d/nvidia-drm-outputclass.conf",
        "*egl/egl_external_platform.d/*nvidia*",
        "*nvidia/nvoptix.bin",
        "*vulkan/icd.d/nvidia_icd*.json",
        "*vulkan/icd.d/nvidia_layers.json",
        "*vulkan/implicit_layer.d/nvidia_layers.json",
        "*vulkansc/icd.d/nvidia_icd*.json",
        "*nvidia.icd",
        "*nvidia.yaml",
        "*nvidia.json",
    ]
    for nvidia_file in _find_files("/run/host/usr", conf_patterns):
        dest_file = str(nvidia_file).replace("/run/host", "", 1)
        dest_file = Path(dest_file)
        if not dest_file.parent.exists():
            try:
                dest_file.parent.mkdir(parents=True, exist_ok=True)
            except Exception:
                print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
                continue
        if not dest_file.parent.stat().st_mode & 0o200:
            print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
            continue
        locked_flags = get_locked_mount_flags(str(nvidia_file))
        flags = "ro" + (f",{locked_flags}" if locked_flags else "")
        mount_bind(str(nvidia_file), str(dest_file), flags)

    # Binaries + Wine/DXVK
    nvidia_binaries = []
    for search_dir in ["/run/host/bin", "/run/host/sbin", "/run/host/usr/bin", "/run/host/usr/sbin"]:
        nvidia_binaries += _find_files(search_dir, ["*nvidia*"])
    nvidia_wine = _find_files("/run/host/usr/lib", ["*nvngx*"])
    for p in Path("/run/host/usr").glob("lib*"):
        if p.is_dir() and p != Path("/run/host/usr/lib"):
            nvidia_wine += _find_files(str(p), ["*nvngx*"])

    for nvidia_file in nvidia_binaries + nvidia_wine:
        dest_file = str(nvidia_file).replace("/run/host", "", 1)
        _mount_nvidia_file(nvidia_file, dest_file)

    # Detect lib dirs
    lib32_dir = "/usr/lib/"
    lib64_dir = "/usr/lib/"
    if Path("/usr/lib/x86_64-linux-gnu").exists():
        lib64_dir = "/usr/lib/x86_64-linux-gnu/"
        lib32_dir = "/usr/lib/i386-linux-gnu/"
    elif Path("/usr/lib64").exists():
        lib64_dir = "/usr/lib64/"
    if Path("/usr/lib32").exists():
        lib32_dir = "/usr/lib32/"

    # .so libraries with path remapping
    lib_patterns = ["*lib*nvidia*.so*", "*nvidia*.so*", "libcuda*.so*", "libnvcuvid*", "libnvoptix*"]
    nvidia_libs = []
    for p in Path("/run/host/usr").glob("lib*"):
        if p.is_dir():
            nvidia_libs += _find_files(str(p), lib_patterns)

    for nvidia_lib in nvidia_libs:
        dest_file = str(nvidia_lib)
        dest_file = dest_file.replace("/run/host/usr/lib/x86_64-linux-gnu/", lib64_dir)
        dest_file = dest_file.replace("/run/host/usr/lib/i386-linux-gnu/", lib32_dir)
        dest_file = dest_file.replace("/run/host/usr/lib64/", lib64_dir)
        dest_file = dest_file.replace("/run/host/usr/lib32/", lib32_dir)
        dest_file = dest_file.replace("/run/host/usr/lib/", lib32_dir)

        if Path(dest_file).exists():
            continue

        dest_file = Path(dest_file)
        if not dest_file.parent.exists():
            try:
                dest_file.parent.mkdir(parents=True, exist_ok=True)
            except Exception:
                print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
                continue
        if not dest_file.parent.stat().st_mode & 0o200:
            print(f"Warning: skpping file {dest_file}, {dest_file.parent} mounted as read-only")
            continue

        nvidia_lib_resolved = Path(nvidia_lib)
        if nvidia_lib_resolved.is_symlink():
            nvidia_lib_resolved = nvidia_lib_resolved.resolve()

        locked_flags = get_locked_mount_flags(str(nvidia_lib_resolved))
        flags = "ro" + (f",{locked_flags}" if locked_flags else "")
        mount_bind(str(nvidia_lib_resolved), str(dest_file), flags)

    # Refresh ldconfig cache
    subprocess.run(["ldconfig"], capture_output=True)


if __name__ == "__main__":
    setup_nvidia()