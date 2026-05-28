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
import subprocess
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))
from mount_utils import init_readlink, mount_bind  # type: ignore

HOST_WATCH = [
    "/etc/hostname",
    "/etc/hosts",
    "/etc/localtime",
    "/etc/resolv.conf",
]


def get_container_id():
    container_id = os.environ.get("CONTAINER_ID", "")

    if Path("/run/.containerenv").exists():
        for line in Path("/run/.containerenv").read_text().splitlines():
            m = re.match(r'^id="(.+)"$', line)
            if m:
                container_id = m.group(1)
                break
    elif Path("/.dockerenv").exists():
        cid = os.environ.get("CONTAINER_ID", "")
        if not cid:
            result = subprocess.run(["hostname"], capture_output=True, text=True)
            cid = result.stdout.strip().split(".")[0]
        result = subprocess.run(
            ["curl", "-s", "--unix-socket", "/run/docker.sock",
             f"http://docker/containers/{cid}/json"],
            capture_output=True, text=True
        )
        m = re.search(r'"Id":"([a-zA-Z0-9]{64})"', result.stdout)
        if m:
            container_id = m.group(1)

    return container_id


def keepalive(container_id):
    Path("/usr/lib/otter/container.ready").touch()
    print("container_setup_done")

    while True:
        for file_watch in HOST_WATCH:
            result = subprocess.run(
                ["findmnt", "-no", "SOURCE", file_watch],
                capture_output=True, text=True
            )
            mount_source = result.stdout.strip()

            if mount_source and container_id not in mount_source:
                file_watch_src = f"/run/host{file_watch}"

                if not Path(file_watch_src).exists():
                    continue

                if Path(file_watch_src).is_symlink():
                    file_watch_src = init_readlink(f"/run/host{file_watch}")
                    if "/run/host" not in file_watch_src:
                        file_watch_src = f"/run/host{file_watch_src}"

                result = subprocess.run(
                    ["diff", file_watch, file_watch_src],
                    capture_output=True
                )
                if result.returncode != 0:
                    subprocess.run(["umount", file_watch])
                    mount_bind(file_watch_src, file_watch)

                    if file_watch == "/etc/hostname":
                        hostname = Path("/etc/hostname").read_text().strip()
                        subprocess.run(["hostname", hostname])

        time.sleep(15)


if __name__ == "__main__":
    container_id = get_container_id()
    keepalive(container_id)